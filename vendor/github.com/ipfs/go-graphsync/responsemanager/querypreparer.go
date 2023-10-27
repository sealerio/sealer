package responsemanager

import (
	"context"
	"math"

	"github.com/ipfs/go-cid"
	ipld "github.com/ipld/go-ipld-prime"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	"github.com/ipld/go-ipld-prime/traversal"
	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/ipfs/go-graphsync"
	"github.com/ipfs/go-graphsync/cidset"
	"github.com/ipfs/go-graphsync/dedupkey"
	"github.com/ipfs/go-graphsync/donotsendfirstblocks"
	"github.com/ipfs/go-graphsync/ipldutil"
	gsmsg "github.com/ipfs/go-graphsync/message"
	"github.com/ipfs/go-graphsync/notifications"
	"github.com/ipfs/go-graphsync/responsemanager/queryexecutor"
	"github.com/ipfs/go-graphsync/responsemanager/responseassembler"
)

type errorString string

func (e errorString) Error() string {
	return string(e)
}

const errInvalidRequest = errorString("request not valid")

type queryPreparer struct {
	requestHooks      RequestHooks
	responseAssembler ResponseAssembler
	linkSystem        ipld.LinkSystem
	// maximum number of links to traverse per request. A value of zero = infinity, or no limit=
	maxLinksPerRequest uint64
}

func (qe *queryPreparer) prepareQuery(ctx context.Context,
	p peer.ID,
	request gsmsg.GraphSyncRequest, signals queryexecutor.ResponseSignals, sub *notifications.TopicDataSubscriber) (ipld.BlockReadOpener, ipldutil.Traverser, bool, error) {
	result := qe.requestHooks.ProcessRequestHooks(p, request)
	var isPaused bool
	failNotifee := notifications.Notifee{Data: graphsync.RequestFailedUnknown, Subscriber: sub}
	rejectNotifee := notifications.Notifee{Data: graphsync.RequestRejected, Subscriber: sub}
	err := qe.responseAssembler.Transaction(p, request.ID(), func(rb responseassembler.ResponseBuilder) error {
		for _, extension := range result.Extensions {
			rb.SendExtensionData(extension)
		}
		if result.Err != nil {
			rb.FinishWithError(graphsync.RequestFailedUnknown)
			rb.AddNotifee(failNotifee)
			return result.Err
		} else if !result.IsValidated {
			rb.FinishWithError(graphsync.RequestRejected)
			rb.AddNotifee(rejectNotifee)
			return errInvalidRequest
		} else if result.IsPaused {
			rb.PauseRequest()
			isPaused = true
		}
		return nil
	})
	if err != nil {
		return nil, nil, false, err
	}
	if err := qe.processDedupByKey(request, p, failNotifee); err != nil {
		return nil, nil, false, err
	}
	if err := qe.processDoNoSendCids(request, p, failNotifee); err != nil {
		return nil, nil, false, err
	}
	if err := qe.processDoNotSendFirstBlocks(request, p, failNotifee); err != nil {
		return nil, nil, false, err
	}
	rootLink := cidlink.Link{Cid: request.Root()}
	linkSystem := result.CustomLinkSystem
	if linkSystem.StorageReadOpener == nil {
		linkSystem = qe.linkSystem
	}
	var budget *traversal.Budget
	if qe.maxLinksPerRequest > 0 {
		budget = &traversal.Budget{
			NodeBudget: math.MaxInt64,
			LinkBudget: int64(qe.maxLinksPerRequest),
		}
	}
	traverser := ipldutil.TraversalBuilder{
		Root:       rootLink,
		Selector:   request.Selector(),
		LinkSystem: linkSystem,
		Chooser:    result.CustomChooser,
		Budget:     budget,
	}.Start(ctx)

	return linkSystem.StorageReadOpener, traverser, isPaused, nil
}

func (qe *queryPreparer) processDedupByKey(request gsmsg.GraphSyncRequest, p peer.ID, failNotifee notifications.Notifee) error {
	dedupData, has := request.Extension(graphsync.ExtensionDeDupByKey)
	if !has {
		return nil
	}
	key, err := dedupkey.DecodeDedupKey(dedupData)
	if err != nil {
		_ = qe.responseAssembler.Transaction(p, request.ID(), func(rb responseassembler.ResponseBuilder) error {
			rb.FinishWithError(graphsync.RequestFailedUnknown)
			rb.AddNotifee(failNotifee)
			return nil
		})
		return err
	}
	qe.responseAssembler.DedupKey(p, request.ID(), key)
	return nil
}

func (qe *queryPreparer) processDoNoSendCids(request gsmsg.GraphSyncRequest, p peer.ID, failNotifee notifications.Notifee) error {
	doNotSendCidsData, has := request.Extension(graphsync.ExtensionDoNotSendCIDs)
	if !has {
		return nil
	}
	cidSet, err := cidset.DecodeCidSet(doNotSendCidsData)
	if err != nil {
		_ = qe.responseAssembler.Transaction(p, request.ID(), func(rb responseassembler.ResponseBuilder) error {
			rb.FinishWithError(graphsync.RequestFailedUnknown)
			rb.AddNotifee(failNotifee)
			return nil
		})
		return err
	}
	links := make([]ipld.Link, 0, cidSet.Len())
	err = cidSet.ForEach(func(c cid.Cid) error {
		links = append(links, cidlink.Link{Cid: c})
		return nil
	})
	if err != nil {
		return err
	}
	qe.responseAssembler.IgnoreBlocks(p, request.ID(), links)
	return nil
}

func (qe *queryPreparer) processDoNotSendFirstBlocks(request gsmsg.GraphSyncRequest, p peer.ID, failNotifee notifications.Notifee) error {
	doNotSendFirstBlocksData, has := request.Extension(graphsync.ExtensionsDoNotSendFirstBlocks)
	if !has {
		return nil
	}
	skipCount, err := donotsendfirstblocks.DecodeDoNotSendFirstBlocks(doNotSendFirstBlocksData)
	if err != nil {
		_ = qe.responseAssembler.Transaction(p, request.ID(), func(rb responseassembler.ResponseBuilder) error {
			rb.FinishWithError(graphsync.RequestFailedUnknown)
			rb.AddNotifee(failNotifee)
			return nil
		})
		return err
	}
	qe.responseAssembler.SkipFirstBlocks(p, request.ID(), skipCount)
	return nil
}
