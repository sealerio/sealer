package requestmanager

import (
	"context"

	"github.com/ipfs/go-graphsync"
)

type responseCollector struct {
	ctx context.Context
}

func newResponseCollector(ctx context.Context) *responseCollector {
	return &responseCollector{ctx}
}

func (rc *responseCollector) collectResponses(
	requestCtx context.Context,
	incomingResponses <-chan graphsync.ResponseProgress,
	incomingErrors <-chan error,
	cancelRequest func(),
	onComplete func(),
) (<-chan graphsync.ResponseProgress, <-chan error) {

	returnedResponses := make(chan graphsync.ResponseProgress)
	returnedErrors := make(chan error)

	go func() {
		var receivedResponses []graphsync.ResponseProgress
		defer close(returnedResponses)
		defer onComplete()
		outgoingResponses := func() chan<- graphsync.ResponseProgress {
			if len(receivedResponses) == 0 {
				return nil
			}
			return returnedResponses
		}
		nextResponse := func() graphsync.ResponseProgress {
			if len(receivedResponses) == 0 {
				return graphsync.ResponseProgress{}
			}
			return receivedResponses[0]
		}
		for len(receivedResponses) > 0 || incomingResponses != nil {
			select {
			case <-rc.ctx.Done():
				return
			case <-requestCtx.Done():
				if incomingResponses != nil {
					cancelRequest()
				}
				return
			case response, ok := <-incomingResponses:
				if !ok {
					incomingResponses = nil
				} else {
					receivedResponses = append(receivedResponses, response)
				}
			case outgoingResponses() <- nextResponse():
				receivedResponses = receivedResponses[1:]
			}
		}
	}()
	go func() {
		var receivedErrors []error
		defer close(returnedErrors)

		outgoingErrors := func() chan<- error {
			if len(receivedErrors) == 0 {
				return nil
			}
			return returnedErrors
		}
		nextError := func() error {
			if len(receivedErrors) == 0 {
				return nil
			}
			return receivedErrors[0]
		}

		for len(receivedErrors) > 0 || incomingErrors != nil {
			select {
			case <-rc.ctx.Done():
				return
			case <-requestCtx.Done():
				select {
				case <-rc.ctx.Done():
				case returnedErrors <- graphsync.RequestClientCancelledErr{}:
				}
				return
			case err, ok := <-incomingErrors:
				if !ok {
					incomingErrors = nil
					// even if the `incomingErrors` channel is closed without any error,
					// the context could still have timed out in which case we need to inform the caller of the same.
					select {
					case <-requestCtx.Done():
						select {
						case <-rc.ctx.Done():
						case returnedErrors <- graphsync.RequestClientCancelledErr{}:
						}
					default:
					}
				} else {
					receivedErrors = append(receivedErrors, err)
				}
			case outgoingErrors() <- nextError():
				receivedErrors = receivedErrors[1:]
			}
		}
	}()
	return returnedResponses, returnedErrors
}
