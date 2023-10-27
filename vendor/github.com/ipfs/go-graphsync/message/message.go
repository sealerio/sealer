package message

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	blocks "github.com/ipfs/go-block-format"
	cid "github.com/ipfs/go-cid"
	"github.com/ipld/go-ipld-prime"
	pool "github.com/libp2p/go-buffer-pool"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-msgio"
	"google.golang.org/protobuf/proto"

	"github.com/ipfs/go-graphsync"
	"github.com/ipfs/go-graphsync/ipldutil"
	pb "github.com/ipfs/go-graphsync/message/pb"
)

// IsTerminalSuccessCode returns true if the response code indicates the
// request terminated successfully.
// DEPRECATED: use status.IsSuccess()
func IsTerminalSuccessCode(status graphsync.ResponseStatusCode) bool {
	return status.IsSuccess()
}

// IsTerminalFailureCode returns true if the response code indicates the
// request terminated in failure.
// DEPRECATED: use status.IsFailure()
func IsTerminalFailureCode(status graphsync.ResponseStatusCode) bool {
	return status.IsFailure()
}

// IsTerminalResponseCode returns true if the response code signals
// the end of the request
// DEPRECATED: use status.IsTerminal()
func IsTerminalResponseCode(status graphsync.ResponseStatusCode) bool {
	return status.IsTerminal()
}

// Exportable is an interface that can serialize to a protobuf
type Exportable interface {
	ToProto() (*pb.Message, error)
	ToNet(w io.Writer) error
}

// GraphSyncRequest is a struct to capture data on a request contained in a
// GraphSyncMessage.
type GraphSyncRequest struct {
	root       cid.Cid
	selector   ipld.Node
	priority   graphsync.Priority
	id         graphsync.RequestID
	extensions map[string][]byte
	isCancel   bool
	isUpdate   bool
}

// GraphSyncResponse is an struct to capture data on a response sent back
// in a GraphSyncMessage.
type GraphSyncResponse struct {
	requestID  graphsync.RequestID
	status     graphsync.ResponseStatusCode
	extensions map[string][]byte
}

type GraphSyncMessage struct {
	requests  map[graphsync.RequestID]GraphSyncRequest
	responses map[graphsync.RequestID]GraphSyncResponse
	blocks    map[cid.Cid]blocks.Block
}

// NewRequest builds a new Graphsync request
func NewRequest(id graphsync.RequestID,
	root cid.Cid,
	selector ipld.Node,
	priority graphsync.Priority,
	extensions ...graphsync.ExtensionData) GraphSyncRequest {

	return newRequest(id, root, selector, priority, false, false, toExtensionsMap(extensions))
}

// CancelRequest request generates a request to cancel an in progress request
func CancelRequest(id graphsync.RequestID) GraphSyncRequest {
	return newRequest(id, cid.Cid{}, nil, 0, true, false, nil)
}

// UpdateRequest generates a new request to update an in progress request with the given extensions
func UpdateRequest(id graphsync.RequestID, extensions ...graphsync.ExtensionData) GraphSyncRequest {
	return newRequest(id, cid.Cid{}, nil, 0, false, true, toExtensionsMap(extensions))
}

func toExtensionsMap(extensions []graphsync.ExtensionData) (extensionsMap map[string][]byte) {
	if len(extensions) > 0 {
		extensionsMap = make(map[string][]byte, len(extensions))
		for _, extension := range extensions {
			extensionsMap[string(extension.Name)] = extension.Data
		}
	}
	return
}

func newRequest(id graphsync.RequestID,
	root cid.Cid,
	selector ipld.Node,
	priority graphsync.Priority,
	isCancel bool,
	isUpdate bool,
	extensions map[string][]byte) GraphSyncRequest {
	return GraphSyncRequest{
		id:         id,
		root:       root,
		selector:   selector,
		priority:   priority,
		isCancel:   isCancel,
		isUpdate:   isUpdate,
		extensions: extensions,
	}
}

// NewResponse builds a new Graphsync response
func NewResponse(requestID graphsync.RequestID,
	status graphsync.ResponseStatusCode,
	extensions ...graphsync.ExtensionData) GraphSyncResponse {
	return newResponse(requestID, status, toExtensionsMap(extensions))
}

func newResponse(requestID graphsync.RequestID,
	status graphsync.ResponseStatusCode, extensions map[string][]byte) GraphSyncResponse {
	return GraphSyncResponse{
		requestID:  requestID,
		status:     status,
		extensions: extensions,
	}
}

func newMessageFromProto(pbm *pb.Message) (GraphSyncMessage, error) {
	requests := make(map[graphsync.RequestID]GraphSyncRequest, len(pbm.GetRequests()))
	for _, req := range pbm.Requests {
		if req == nil {
			return GraphSyncMessage{}, errors.New("request is nil")
		}
		var root cid.Cid
		var err error
		if !req.Cancel && !req.Update {
			root, err = cid.Cast(req.Root)
			if err != nil {
				return GraphSyncMessage{}, err
			}
		}

		var selector ipld.Node
		if !req.Cancel && !req.Update {
			selector, err = ipldutil.DecodeNode(req.Selector)
			if err != nil {
				return GraphSyncMessage{}, err
			}
		}
		exts := req.GetExtensions()
		if exts == nil {
			exts = make(map[string][]byte)
		}
		requests[graphsync.RequestID(req.Id)] = newRequest(graphsync.RequestID(req.Id), root, selector, graphsync.Priority(req.Priority), req.Cancel, req.Update, exts)
	}

	responses := make(map[graphsync.RequestID]GraphSyncResponse, len(pbm.GetResponses()))
	for _, res := range pbm.Responses {
		if res == nil {
			return GraphSyncMessage{}, errors.New("response is nil")
		}
		exts := res.GetExtensions()
		if exts == nil {
			exts = make(map[string][]byte)
		}
		responses[graphsync.RequestID(res.Id)] = newResponse(graphsync.RequestID(res.Id), graphsync.ResponseStatusCode(res.Status), exts)
	}

	blks := make(map[cid.Cid]blocks.Block, len(pbm.GetData()))
	for _, b := range pbm.GetData() {
		if b == nil {
			return GraphSyncMessage{}, errors.New("block is nil")
		}

		pref, err := cid.PrefixFromBytes(b.GetPrefix())
		if err != nil {
			return GraphSyncMessage{}, err
		}

		c, err := pref.Sum(b.GetData())
		if err != nil {
			return GraphSyncMessage{}, err
		}

		blk, err := blocks.NewBlockWithCid(b.GetData(), c)
		if err != nil {
			return GraphSyncMessage{}, err
		}

		blks[blk.Cid()] = blk
	}

	return GraphSyncMessage{
		requests, responses, blks,
	}, nil
}

func (gsm GraphSyncMessage) Empty() bool {
	return len(gsm.blocks) == 0 && len(gsm.requests) == 0 && len(gsm.responses) == 0
}

func (gsm GraphSyncMessage) Requests() []GraphSyncRequest {
	requests := make([]GraphSyncRequest, 0, len(gsm.requests))
	for _, request := range gsm.requests {
		requests = append(requests, request)
	}
	return requests
}

func (gsm GraphSyncMessage) Responses() []GraphSyncResponse {
	responses := make([]GraphSyncResponse, 0, len(gsm.responses))
	for _, response := range gsm.responses {
		responses = append(responses, response)
	}
	return responses
}

func (gsm GraphSyncMessage) Blocks() []blocks.Block {
	bs := make([]blocks.Block, 0, len(gsm.blocks))
	for _, block := range gsm.blocks {
		bs = append(bs, block)
	}
	return bs
}

// FromNet can read a network stream to deserialized a GraphSyncMessage
func FromNet(r io.Reader) (GraphSyncMessage, error) {
	reader := msgio.NewVarintReaderSize(r, network.MessageSizeMax)
	return FromMsgReader(reader)
}

// FromMsgReader can deserialize a protobuf message into a GraphySyncMessage.
func FromMsgReader(r msgio.Reader) (GraphSyncMessage, error) {
	msg, err := r.ReadMsg()
	if err != nil {
		return GraphSyncMessage{}, err
	}

	var pb pb.Message
	err = proto.Unmarshal(msg, &pb)
	r.ReleaseMsg(msg)
	if err != nil {
		return GraphSyncMessage{}, err
	}

	return newMessageFromProto(&pb)
}

func (gsm GraphSyncMessage) ToProto() (*pb.Message, error) {
	pbm := new(pb.Message)
	pbm.Requests = make([]*pb.Message_Request, 0, len(gsm.requests))
	for _, request := range gsm.requests {
		var selector []byte
		var err error
		if request.selector != nil {
			selector, err = ipldutil.EncodeNode(request.selector)
			if err != nil {
				return nil, err
			}
		}
		pbm.Requests = append(pbm.Requests, &pb.Message_Request{
			Id:         int32(request.id),
			Root:       request.root.Bytes(),
			Selector:   selector,
			Priority:   int32(request.priority),
			Cancel:     request.isCancel,
			Update:     request.isUpdate,
			Extensions: request.extensions,
		})
	}

	pbm.Responses = make([]*pb.Message_Response, 0, len(gsm.responses))
	for _, response := range gsm.responses {
		pbm.Responses = append(pbm.Responses, &pb.Message_Response{
			Id:         int32(response.requestID),
			Status:     int32(response.status),
			Extensions: response.extensions,
		})
	}

	blocks := gsm.Blocks()
	pbm.Data = make([]*pb.Message_Block, 0, len(blocks))
	for _, b := range blocks {
		pbm.Data = append(pbm.Data, &pb.Message_Block{
			Data:   b.RawData(),
			Prefix: b.Cid().Prefix().Bytes(),
		})
	}
	return pbm, nil
}

func (gsm GraphSyncMessage) ToNet(w io.Writer) error {
	msg, err := gsm.ToProto()
	if err != nil {
		return err
	}
	size := proto.Size(msg)
	buf := pool.Get(size + binary.MaxVarintLen64)
	defer pool.Put(buf)

	n := binary.PutUvarint(buf, uint64(size))

	out, err := proto.MarshalOptions{}.MarshalAppend(buf[:n], msg)
	if err != nil {
		return err
	}
	_, err = w.Write(out)
	return err
}

func (gsm GraphSyncMessage) Loggable() map[string]interface{} {
	requests := make([]string, 0, len(gsm.requests))
	for _, request := range gsm.requests {
		requests = append(requests, fmt.Sprintf("%d", request.id))
	}
	responses := make([]string, 0, len(gsm.responses))
	for _, response := range gsm.responses {
		responses = append(responses, fmt.Sprintf("%d", response.requestID))
	}
	return map[string]interface{}{
		"requests":  requests,
		"responses": responses,
	}
}

func (gsm GraphSyncMessage) Clone() GraphSyncMessage {
	requests := make(map[graphsync.RequestID]GraphSyncRequest, len(gsm.requests))
	for id, request := range gsm.requests {
		requests[id] = request
	}
	responses := make(map[graphsync.RequestID]GraphSyncResponse, len(gsm.responses))
	for id, response := range gsm.responses {
		responses[id] = response
	}
	blocks := make(map[cid.Cid]blocks.Block, len(gsm.blocks))
	for cid, block := range gsm.blocks {
		blocks[cid] = block
	}
	return GraphSyncMessage{requests, responses, blocks}
}

// ID Returns the request ID for this Request
func (gsr GraphSyncRequest) ID() graphsync.RequestID { return gsr.id }

// Root returns the CID to the root block of this request
func (gsr GraphSyncRequest) Root() cid.Cid { return gsr.root }

// Selector returns the byte representation of the selector for this request
func (gsr GraphSyncRequest) Selector() ipld.Node { return gsr.selector }

// Priority returns the priority of this request
func (gsr GraphSyncRequest) Priority() graphsync.Priority { return gsr.priority }

// Extension returns the content for an extension on a response, or errors
// if extension is not present
func (gsr GraphSyncRequest) Extension(name graphsync.ExtensionName) ([]byte, bool) {
	if gsr.extensions == nil {
		return nil, false
	}
	val, ok := gsr.extensions[string(name)]
	if !ok {
		return nil, false
	}
	return val, true
}

// IsCancel returns true if this particular request is being cancelled
func (gsr GraphSyncRequest) IsCancel() bool { return gsr.isCancel }

// IsUpdate returns true if this particular request is being updated
func (gsr GraphSyncRequest) IsUpdate() bool { return gsr.isUpdate }

// RequestID returns the request ID for this response
func (gsr GraphSyncResponse) RequestID() graphsync.RequestID { return gsr.requestID }

// Status returns the status for a response
func (gsr GraphSyncResponse) Status() graphsync.ResponseStatusCode { return gsr.status }

// Extension returns the content for an extension on a response, or errors
// if extension is not present
func (gsr GraphSyncResponse) Extension(name graphsync.ExtensionName) ([]byte, bool) {
	if gsr.extensions == nil {
		return nil, false
	}
	val, ok := gsr.extensions[string(name)]
	if !ok {
		return nil, false
	}
	return val, true

}

// ReplaceExtensions merges the extensions given extensions into the request to create a new request,
// but always uses new data
func (gsr GraphSyncRequest) ReplaceExtensions(extensions []graphsync.ExtensionData) GraphSyncRequest {
	req, _ := gsr.MergeExtensions(extensions, func(name graphsync.ExtensionName, oldData []byte, newData []byte) ([]byte, error) {
		return newData, nil
	})
	return req
}

// MergeExtensions merges the given list of extensions to produce a new request with the combination of the old request
// plus the new extensions. When an old extension and a new extension are both present, mergeFunc is called to produce
// the result
func (gsr GraphSyncRequest) MergeExtensions(extensions []graphsync.ExtensionData, mergeFunc func(name graphsync.ExtensionName, oldData []byte, newData []byte) ([]byte, error)) (GraphSyncRequest, error) {
	if gsr.extensions == nil {
		return newRequest(gsr.id, gsr.root, gsr.selector, gsr.priority, gsr.isCancel, gsr.isUpdate, toExtensionsMap(extensions)), nil
	}
	newExtensionMap := toExtensionsMap(extensions)
	combinedExtensions := make(map[string][]byte)
	for name, newData := range newExtensionMap {
		oldData, ok := gsr.extensions[name]
		if !ok {
			combinedExtensions[name] = newData
			continue
		}
		resultData, err := mergeFunc(graphsync.ExtensionName(name), oldData, newData)
		if err != nil {
			return GraphSyncRequest{}, err
		}
		combinedExtensions[name] = resultData
	}

	for name, oldData := range gsr.extensions {
		_, ok := combinedExtensions[name]
		if ok {
			continue
		}
		combinedExtensions[name] = oldData
	}
	return newRequest(gsr.id, gsr.root, gsr.selector, gsr.priority, gsr.isCancel, gsr.isUpdate, combinedExtensions), nil
}
