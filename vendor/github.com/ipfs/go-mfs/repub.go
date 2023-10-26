package mfs

import (
	"context"
	"time"

	cid "github.com/ipfs/go-cid"
)

// PubFunc is the user-defined function that determines exactly what
// logic entails "publishing" a `Cid` value.
type PubFunc func(context.Context, cid.Cid) error

// Republisher manages when to publish a given entry.
type Republisher struct {
	TimeoutLong  time.Duration
	TimeoutShort time.Duration
	RetryTimeout time.Duration
	pubfunc      PubFunc

	update           chan cid.Cid
	immediatePublish chan chan struct{}

	ctx    context.Context
	cancel func()
}

// NewRepublisher creates a new Republisher object to republish the given root
// using the given short and long time intervals.
func NewRepublisher(ctx context.Context, pf PubFunc, tshort, tlong time.Duration) *Republisher {
	ctx, cancel := context.WithCancel(ctx)
	return &Republisher{
		TimeoutShort:     tshort,
		TimeoutLong:      tlong,
		RetryTimeout:     tlong,
		update:           make(chan cid.Cid, 1),
		pubfunc:          pf,
		immediatePublish: make(chan chan struct{}),
		ctx:              ctx,
		cancel:           cancel,
	}
}

// WaitPub waits for the current value to be published (or returns early
// if it already has).
func (rp *Republisher) WaitPub(ctx context.Context) error {
	wait := make(chan struct{})
	select {
	case rp.immediatePublish <- wait:
	case <-ctx.Done():
		return ctx.Err()
	}
	select {
	case <-wait:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (rp *Republisher) Close() error {
	// TODO(steb): Wait for `Run` to stop
	err := rp.WaitPub(rp.ctx)
	rp.cancel()
	return err
}

// Update the current value. The value will be published after a delay but each
// consecutive call to Update may extend this delay up to TimeoutLong.
func (rp *Republisher) Update(c cid.Cid) {
	select {
	case <-rp.update:
		select {
		case rp.update <- c:
		default:
			// Don't try again. If we hit this case, there's a
			// concurrent publish and we can safely let that
			// concurrent publish win.
		}
	case rp.update <- c:
	}
}

// Run contains the core logic of the `Republisher`. It calls the user-defined
// `pubfunc` function whenever the `Cid` value is updated to a *new* value. The
// complexity comes from the fact that `pubfunc` may be slow so we need to batch
// updates.
//
// Algorithm:
//   1. When we receive the first update after publishing, we set a `longer` timer.
//   2. When we receive any update, we reset the `quick` timer.
//   3. If either the `quick` timeout or the `longer` timeout elapses,
//      we call `publish` with the latest updated value.
//
// The `longer` timer ensures that we delay publishing by at most
// `TimeoutLong`. The `quick` timer allows us to publish sooner if
// it looks like there are no more updates coming down the pipe.
//
// Note: If a publish fails, we retry repeatedly every TimeoutRetry.
func (rp *Republisher) Run(lastPublished cid.Cid) {
	quick := time.NewTimer(0)
	if !quick.Stop() {
		<-quick.C
	}
	longer := time.NewTimer(0)
	if !longer.Stop() {
		<-longer.C
	}

	var toPublish cid.Cid
	for rp.ctx.Err() == nil {
		var waiter chan struct{}

		select {
		case <-rp.ctx.Done():
			return
		case newValue := <-rp.update:
			// Skip already published values.
			if lastPublished.Equals(newValue) {
				// Break to the end of the switch to cleanup any
				// timers.
				toPublish = cid.Undef
				break
			}

			// If we aren't already waiting to publish something,
			// reset the long timeout.
			if !toPublish.Defined() {
				longer.Reset(rp.TimeoutLong)
			}

			// Always reset the short timeout.
			quick.Reset(rp.TimeoutShort)

			// Finally, set the new value to publish.
			toPublish = newValue
			continue
		case waiter = <-rp.immediatePublish:
			// Make sure to grab the *latest* value to publish.
			select {
			case toPublish = <-rp.update:
			default:
			}

			// Avoid publishing duplicate values
			if lastPublished.Equals(toPublish) {
				toPublish = cid.Undef
			}
		case <-quick.C:
		case <-longer.C:
		}

		// Cleanup, publish, and close waiters.

		// 1. Stop any timers. Don't use the `if !t.Stop() { ... }`
		//    idiom as these timers may not be running.

		quick.Stop()
		select {
		case <-quick.C:
		default:
		}

		longer.Stop()
		select {
		case <-longer.C:
		default:
		}

		// 2. If we have a value to publish, publish it now.
		if toPublish.Defined() {
			for {
				err := rp.pubfunc(rp.ctx, toPublish)
				if err == nil {
					break
				}
				// Keep retrying until we succeed or we abort.
				// TODO(steb): We could try pulling new values
				// off `update` but that's not critical (and
				// complicates this code a bit). We'll pull off
				// a new value on the next loop through.
				select {
				case <-time.After(rp.RetryTimeout):
				case <-rp.ctx.Done():
					return
				}
			}
			lastPublished = toPublish
			toPublish = cid.Undef
		}

		// 3. Trigger anything waiting in `WaitPub`.
		if waiter != nil {
			close(waiter)
		}
	}
}
