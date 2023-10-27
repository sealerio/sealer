// Package simple implements structures and methods to provide blocks,
// keep track of which blocks are provided, and to allow those blocks to
// be reprovided.
package simple

import (
	"context"
	"time"

	"github.com/ipfs/go-cid"
	q "github.com/ipfs/go-ipfs-provider/queue"
	logging "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p-core/routing"
)

var logP = logging.Logger("provider.simple")

// Provider announces blocks to the network
type Provider struct {
	ctx context.Context
	// the CIDs for which provide announcements should be made
	queue *q.Queue
	// used to announce providing to the network
	contentRouting routing.ContentRouting
	// how long to wait for announce to complete before giving up
	timeout time.Duration
	// how many workers concurrently work through thhe queue
	workerLimit int
}

// Option defines the functional option type that can be used to configure
// provider instances
type Option func(*Provider)

// WithTimeout is an option to set a timeout on a provider
func WithTimeout(timeout time.Duration) Option {
	return func(p *Provider) {
		p.timeout = timeout
	}
}

// MaxWorkers is an option to set the max workers on a provider
func MaxWorkers(count int) Option {
	return func(p *Provider) {
		p.workerLimit = count
	}
}

// NewProvider creates a provider that announces blocks to the network using a content router
func NewProvider(ctx context.Context, queue *q.Queue, contentRouting routing.ContentRouting, options ...Option) *Provider {
	p := &Provider{
		ctx:            ctx,
		queue:          queue,
		contentRouting: contentRouting,
		workerLimit:    8,
	}

	for _, option := range options {
		option(p)
	}

	return p
}

// Close stops the provider
func (p *Provider) Close() error {
	return p.queue.Close()
}

// Run workers to handle provide requests.
func (p *Provider) Run() {
	p.handleAnnouncements()
}

// Provide the given cid using specified strategy.
func (p *Provider) Provide(root cid.Cid) error {
	return p.queue.Enqueue(root)
}

// Handle all outgoing cids by providing (announcing) them
func (p *Provider) handleAnnouncements() {
	for workers := 0; workers < p.workerLimit; workers++ {
		go func() {
			for p.ctx.Err() == nil {
				select {
				case <-p.ctx.Done():
					return
				case c, ok := <-p.queue.Dequeue():
					if !ok {
						// queue closed.
						return
					}

					p.doProvide(c)
				}
			}
		}()
	}
}

func (p *Provider) doProvide(c cid.Cid) {
	ctx := p.ctx
	if p.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, p.timeout)
		defer cancel()
	} else {
		ctx = p.ctx
	}

	logP.Info("announce - start - ", c)
	if err := p.contentRouting.Provide(ctx, c, true); err != nil {
		logP.Warnf("Unable to provide entry: %s, %s", c, err)
	}
	logP.Info("announce - end - ", c)
}
