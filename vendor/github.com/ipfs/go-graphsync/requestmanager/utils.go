package requestmanager

import (
	"github.com/ipfs/go-graphsync"
	gsmsg "github.com/ipfs/go-graphsync/message"
	"github.com/ipfs/go-graphsync/metadata"
)

func metadataForResponses(responses []gsmsg.GraphSyncResponse) map[graphsync.RequestID]metadata.Metadata {
	responseMetadata := make(map[graphsync.RequestID]metadata.Metadata, len(responses))
	for _, response := range responses {
		mdRaw, found := response.Extension(graphsync.ExtensionMetadata)
		if !found {
			log.Warnf("Unable to decode metadata in response for request id: %d", response.RequestID())
			continue
		}
		md, err := metadata.DecodeMetadata(mdRaw)
		if err != nil {
			log.Warnf("Unable to decode metadata in response for request id: %d", response.RequestID())
			continue
		}
		responseMetadata[response.RequestID()] = md
	}
	return responseMetadata
}
