package message

import (
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	"github.com/ipld/go-ipld-prime"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"

	"github.com/ipfs/go-graphsync"
	"github.com/ipfs/go-graphsync/metadata"
)

// Builder captures components of a message across multiple
// requests for a given peer and then generates the corresponding
// GraphSync message when ready to send
type Builder struct {
	topic              Topic
	outgoingBlocks     map[cid.Cid]blocks.Block
	blkSize            uint64
	completedResponses map[graphsync.RequestID]graphsync.ResponseStatusCode
	outgoingResponses  map[graphsync.RequestID]metadata.Metadata
	extensions         map[graphsync.RequestID][]graphsync.ExtensionData
	requests           map[graphsync.RequestID]GraphSyncRequest
}

// Topic is an identifier for notifications about this response builder
type Topic uint64

// NewBuilder generates a new Builder.
func NewBuilder(topic Topic) *Builder {
	return &Builder{
		topic:              topic,
		requests:           make(map[graphsync.RequestID]GraphSyncRequest),
		outgoingBlocks:     make(map[cid.Cid]blocks.Block),
		completedResponses: make(map[graphsync.RequestID]graphsync.ResponseStatusCode),
		outgoingResponses:  make(map[graphsync.RequestID]metadata.Metadata),
		extensions:         make(map[graphsync.RequestID][]graphsync.ExtensionData),
	}
}

// AddRequest registers a new request to be added to the message.
func (b *Builder) AddRequest(request GraphSyncRequest) {
	b.requests[request.ID()] = request
}

// AddBlock adds the given block to the message.
func (b *Builder) AddBlock(block blocks.Block) {
	b.blkSize += uint64(len(block.RawData()))
	b.outgoingBlocks[block.Cid()] = block
}

// AddExtensionData adds the given extension data to to the message
func (b *Builder) AddExtensionData(requestID graphsync.RequestID, extension graphsync.ExtensionData) {
	b.extensions[requestID] = append(b.extensions[requestID], extension)
	// make sure this extension goes out in next response even if no links are sent
	_, ok := b.outgoingResponses[requestID]
	if !ok {
		b.outgoingResponses[requestID] = nil
	}
}

// BlockSize returns the total size of all blocks in this message
func (b *Builder) BlockSize() uint64 {
	return b.blkSize
}

// AddLink adds the given link and whether its block is present
// to the message for the given request ID.
func (b *Builder) AddLink(requestID graphsync.RequestID, link ipld.Link, blockPresent bool) {
	b.outgoingResponses[requestID] = append(b.outgoingResponses[requestID], metadata.Item{Link: link.(cidlink.Link).Cid, BlockPresent: blockPresent})
}

// AddResponseCode marks the given request as completed in the message,
// as well as whether the graphsync request responded with complete or partial
// data.
func (b *Builder) AddResponseCode(requestID graphsync.RequestID, status graphsync.ResponseStatusCode) {
	b.completedResponses[requestID] = status
	// make sure this completion goes out in next response even if no links are sent
	_, ok := b.outgoingResponses[requestID]
	if !ok {
		b.outgoingResponses[requestID] = nil
	}
}

// Empty returns true if there is no content to send
func (b *Builder) Empty() bool {
	return len(b.requests) == 0 && len(b.outgoingBlocks) == 0 && len(b.outgoingResponses) == 0
}

// Build assembles and encodes message data from the added requests, links, and blocks.
func (b *Builder) Build() (GraphSyncMessage, error) {
	responses := make(map[graphsync.RequestID]GraphSyncResponse, len(b.outgoingResponses))
	for requestID, linkMap := range b.outgoingResponses {
		mdRaw, err := metadata.EncodeMetadata(linkMap)
		if err != nil {
			return GraphSyncMessage{}, err
		}
		b.extensions[requestID] = append(b.extensions[requestID], graphsync.ExtensionData{
			Name: graphsync.ExtensionMetadata,
			Data: mdRaw,
		})
		status, isComplete := b.completedResponses[requestID]
		responses[requestID] = NewResponse(requestID, responseCode(status, isComplete), b.extensions[requestID]...)
	}
	return GraphSyncMessage{
		b.requests, responses, b.outgoingBlocks,
	}, nil
}

// Topic returns the identifier for notifications sent about this builder
func (b *Builder) Topic() Topic {
	return b.topic
}

func responseCode(status graphsync.ResponseStatusCode, isComplete bool) graphsync.ResponseStatusCode {
	if !isComplete {
		return graphsync.PartialResponse
	}
	return status
}
