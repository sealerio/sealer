// Package measure provides a Datastore wrapper that records metrics
// using github.com/ipfs/go-metrics-interface
package measure

import (
	"context"
	"io"
	"time"

	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	"github.com/ipfs/go-metrics-interface"
)

var (
	// sort latencies in buckets with following upper bounds in seconds
	datastoreLatencyBuckets = []float64{1e-4, 1e-3, 1e-2, 1e-1, 1}

	// sort sizes in buckets with following upper bounds in bytes
	datastoreSizeBuckets = []float64{1 << 6, 1 << 12, 1 << 18, 1 << 24}
)

// New wraps the datastore, providing metrics on the operations. The
// metrics are registered with names starting with prefix and a dot.
func New(prefix string, ds datastore.Datastore) *measure {
	m := &measure{
		backend: ds,

		putNum: metrics.New(prefix+".put_total", "Total number of Datastore.Put calls").Counter(),
		putErr: metrics.New(prefix+".put.errors_total", "Number of errored Datastore.Put calls").Counter(),
		putLatency: metrics.New(prefix+".put.latency_seconds",
			"Latency distribution of Datastore.Put calls").Histogram(datastoreLatencyBuckets),
		putSize: metrics.New(prefix+".put.size_bytes",
			"Size distribution of stored byte slices").Histogram(datastoreSizeBuckets),

		syncNum: metrics.New(prefix+".sync_total", "Total number of Datastore.Sync calls").Counter(),
		syncErr: metrics.New(prefix+".sync.errors_total", "Number of errored Datastore.Sync calls").Counter(),
		syncLatency: metrics.New(prefix+".sync.latency_seconds",
			"Latency distribution of Datastore.Sync calls").Histogram(datastoreLatencyBuckets),

		getNum: metrics.New(prefix+".get_total", "Total number of Datastore.Get calls").Counter(),
		getErr: metrics.New(prefix+".get.errors_total", "Number of errored Datastore.Get calls").Counter(),
		getLatency: metrics.New(prefix+".get.latency_seconds",
			"Latency distribution of Datastore.Get calls").Histogram(datastoreLatencyBuckets),
		getSize: metrics.New(prefix+".get.size_bytes",
			"Size distribution of retrieved byte slices").Histogram(datastoreSizeBuckets),

		hasNum: metrics.New(prefix+".has_total", "Total number of Datastore.Has calls").Counter(),
		hasErr: metrics.New(prefix+".has.errors_total", "Number of errored Datastore.Has calls").Counter(),
		hasLatency: metrics.New(prefix+".has.latency_seconds",
			"Latency distribution of Datastore.Has calls").Histogram(datastoreLatencyBuckets),
		getsizeNum: metrics.New(prefix+".getsize_total", "Total number of Datastore.GetSize calls").Counter(),
		getsizeErr: metrics.New(prefix+".getsize.errors_total", "Number of errored Datastore.GetSize calls").Counter(),
		getsizeLatency: metrics.New(prefix+".getsize.latency_seconds",
			"Latency distribution of Datastore.GetSize calls").Histogram(datastoreLatencyBuckets),

		deleteNum: metrics.New(prefix+".delete_total", "Total number of Datastore.Delete calls").Counter(),
		deleteErr: metrics.New(prefix+".delete.errors_total", "Number of errored Datastore.Delete calls").Counter(),
		deleteLatency: metrics.New(prefix+".delete.latency_seconds",
			"Latency distribution of Datastore.Delete calls").Histogram(datastoreLatencyBuckets),

		queryNum: metrics.New(prefix+".query_total", "Total number of Datastore.Query calls").Counter(),
		queryErr: metrics.New(prefix+".query.errors_total", "Number of errored Datastore.Query calls").Counter(),
		queryLatency: metrics.New(prefix+".query.latency_seconds",
			"Latency distribution of Datastore.Query calls").Histogram(datastoreLatencyBuckets),

		checkNum: metrics.New(prefix+".check_total", "Total number of Datastore.Check calls").Counter(),
		checkErr: metrics.New(prefix+".check.errors_total", "Number of errored Datastore.Check calls").Counter(),
		checkLatency: metrics.New(prefix+".check.latency_seconds",
			"Latency distribution of Datastore.Check calls").Histogram(datastoreLatencyBuckets),

		scrubNum: metrics.New(prefix+".scrub_total", "Total number of Datastore.Scrub calls").Counter(),
		scrubErr: metrics.New(prefix+".scrub.errors_total", "Number of errored Datastore.Scrub calls").Counter(),
		scrubLatency: metrics.New(prefix+".scrub.latency_seconds",
			"Latency distribution of Datastore.Scrub calls").Histogram(datastoreLatencyBuckets),

		gcNum: metrics.New(prefix+".gc_total", "Total number of Datastore.CollectGarbage calls").Counter(),
		gcErr: metrics.New(prefix+".gc.errors_total", "Number of errored Datastore.CollectGarbage calls").Counter(),
		gcLatency: metrics.New(prefix+".gc.latency_seconds",
			"Latency distribution of Datastore.CollectGarbage calls").Histogram(datastoreLatencyBuckets),

		duNum: metrics.New(prefix+".du_total", "Total number of Datastore.DiskUsage calls").Counter(),
		duErr: metrics.New(prefix+".du.errors_total", "Number of errored Datastore.DiskUsage calls").Counter(),
		duLatency: metrics.New(prefix+".du.latency_seconds",
			"Latency distribution of Datastore.DiskUsage calls").Histogram(datastoreLatencyBuckets),

		batchPutNum: metrics.New(prefix+".batchput_total", "Total number of Batch.Put calls").Counter(),
		batchPutErr: metrics.New(prefix+".batchput.errors_total", "Number of errored Batch.Put calls").Counter(),
		batchPutLatency: metrics.New(prefix+".batchput.latency_seconds",
			"Latency distribution of Batch.Put calls").Histogram(datastoreLatencyBuckets),
		batchPutSize: metrics.New(prefix+".batchput.size_bytes",
			"Size distribution of byte slices put into batches").Histogram(datastoreSizeBuckets),

		batchDeleteNum: metrics.New(prefix+".batchdelete_total", "Total number of Batch.Delete calls").Counter(),
		batchDeleteErr: metrics.New(prefix+".batchdelete.errors_total", "Number of errored Batch.Delete calls").Counter(),
		batchDeleteLatency: metrics.New(prefix+".batchdelete.latency_seconds",
			"Latency distribution of Batch.Delete calls").Histogram(datastoreLatencyBuckets),

		batchCommitNum: metrics.New(prefix+".batchcommit_total", "Total number of Batch.Commit calls").Counter(),
		batchCommitErr: metrics.New(prefix+".batchcommit.errors_total", "Number of errored Batch.Commit calls").Counter(),
		batchCommitLatency: metrics.New(prefix+".batchcommit.latency_seconds",
			"Latency distribution of Batch.Commit calls").Histogram(datastoreLatencyBuckets),
	}
	return m
}

type measure struct {
	backend datastore.Datastore

	putNum     metrics.Counter
	putErr     metrics.Counter
	putLatency metrics.Histogram
	putSize    metrics.Histogram

	syncNum     metrics.Counter
	syncErr     metrics.Counter
	syncLatency metrics.Histogram

	getNum     metrics.Counter
	getErr     metrics.Counter
	getLatency metrics.Histogram
	getSize    metrics.Histogram

	hasNum     metrics.Counter
	hasErr     metrics.Counter
	hasLatency metrics.Histogram

	getsizeNum     metrics.Counter
	getsizeErr     metrics.Counter
	getsizeLatency metrics.Histogram

	deleteNum     metrics.Counter
	deleteErr     metrics.Counter
	deleteLatency metrics.Histogram

	queryNum     metrics.Counter
	queryErr     metrics.Counter
	queryLatency metrics.Histogram

	checkNum     metrics.Counter
	checkErr     metrics.Counter
	checkLatency metrics.Histogram

	scrubNum     metrics.Counter
	scrubErr     metrics.Counter
	scrubLatency metrics.Histogram

	gcNum     metrics.Counter
	gcErr     metrics.Counter
	gcLatency metrics.Histogram

	duNum     metrics.Counter
	duErr     metrics.Counter
	duLatency metrics.Histogram

	batchPutNum     metrics.Counter
	batchPutErr     metrics.Counter
	batchPutLatency metrics.Histogram
	batchPutSize    metrics.Histogram

	batchDeleteNum     metrics.Counter
	batchDeleteErr     metrics.Counter
	batchDeleteLatency metrics.Histogram

	batchCommitNum     metrics.Counter
	batchCommitErr     metrics.Counter
	batchCommitLatency metrics.Histogram
}

func recordLatency(h metrics.Histogram, start time.Time) {
	elapsed := time.Since(start)
	h.Observe(elapsed.Seconds())
}

func (m *measure) Put(ctx context.Context, key datastore.Key, value []byte) error {
	defer recordLatency(m.putLatency, time.Now())
	m.putNum.Inc()
	m.putSize.Observe(float64(len(value)))
	err := m.backend.Put(ctx, key, value)
	if err != nil {
		m.putErr.Inc()
	}
	return err
}

func (m *measure) Sync(ctx context.Context, prefix datastore.Key) error {
	defer recordLatency(m.syncLatency, time.Now())
	m.syncNum.Inc()
	err := m.backend.Sync(ctx, prefix)
	if err != nil {
		m.syncErr.Inc()
	}
	return err
}

func (m *measure) Get(ctx context.Context, key datastore.Key) (value []byte, err error) {
	defer recordLatency(m.getLatency, time.Now())
	m.getNum.Inc()
	value, err = m.backend.Get(ctx, key)
	switch err {
	case nil:
		m.getSize.Observe(float64(len(value)))
	case datastore.ErrNotFound:
		// Not really an error.
	default:
		m.getErr.Inc()
	}
	return value, err
}

func (m *measure) Has(ctx context.Context, key datastore.Key) (exists bool, err error) {
	defer recordLatency(m.hasLatency, time.Now())
	m.hasNum.Inc()
	exists, err = m.backend.Has(ctx, key)
	if err != nil {
		m.hasErr.Inc()
	}
	return exists, err
}

func (m *measure) GetSize(ctx context.Context, key datastore.Key) (size int, err error) {
	defer recordLatency(m.getsizeLatency, time.Now())
	m.getsizeNum.Inc()
	size, err = m.backend.GetSize(ctx, key)
	switch err {
	case nil, datastore.ErrNotFound:
		// Not really an error.
	default:
		m.getsizeErr.Inc()
	}
	return size, err
}

func (m *measure) Delete(ctx context.Context, key datastore.Key) error {
	defer recordLatency(m.deleteLatency, time.Now())
	m.deleteNum.Inc()
	err := m.backend.Delete(ctx, key)
	if err != nil {
		m.deleteErr.Inc()
	}
	return err
}

func (m *measure) Query(ctx context.Context, q query.Query) (query.Results, error) {
	defer recordLatency(m.queryLatency, time.Now())
	m.queryNum.Inc()
	res, err := m.backend.Query(ctx, q)
	if err != nil {
		m.queryErr.Inc()
	}
	return res, err
}

func (m *measure) Check(ctx context.Context) error {
	defer recordLatency(m.checkLatency, time.Now())
	m.checkNum.Inc()
	if c, ok := m.backend.(datastore.CheckedDatastore); ok {
		err := c.Check(ctx)
		if err != nil {
			m.checkErr.Inc()
		}
		return err
	}
	return nil
}

func (m *measure) Scrub(ctx context.Context) error {
	defer recordLatency(m.scrubLatency, time.Now())
	m.scrubNum.Inc()
	if c, ok := m.backend.(datastore.ScrubbedDatastore); ok {
		err := c.Scrub(ctx)
		if err != nil {
			m.scrubErr.Inc()
		}
		return err
	}
	return nil
}

func (m *measure) CollectGarbage(ctx context.Context) error {
	defer recordLatency(m.gcLatency, time.Now())
	m.gcNum.Inc()
	if c, ok := m.backend.(datastore.GCDatastore); ok {
		err := c.CollectGarbage(ctx)
		if err != nil {
			m.gcErr.Inc()
		}
		return err
	}
	return nil
}

func (m *measure) DiskUsage(ctx context.Context) (uint64, error) {
	defer recordLatency(m.duLatency, time.Now())
	m.duNum.Inc()
	size, err := datastore.DiskUsage(ctx, m.backend)
	if err != nil {
		m.duErr.Inc()
	}
	return size, err
}

type measuredBatch struct {
	b datastore.Batch
	m *measure
}

func (m *measure) Batch(ctx context.Context) (datastore.Batch, error) {
	bds, ok := m.backend.(datastore.Batching)
	if !ok {
		return nil, datastore.ErrBatchUnsupported
	}
	batch, err := bds.Batch(ctx)
	if err != nil {
		return nil, err
	}

	return &measuredBatch{
		b: batch,
		m: m,
	}, nil
}

func (mt *measuredBatch) Put(ctx context.Context, key datastore.Key, val []byte) error {
	defer recordLatency(mt.m.batchPutLatency, time.Now())
	mt.m.batchPutNum.Inc()
	mt.m.batchPutSize.Observe(float64(len(val)))
	err := mt.b.Put(ctx, key, val)
	if err != nil {
		mt.m.batchPutErr.Inc()
	}
	return err
}

func (mt *measuredBatch) Delete(ctx context.Context, key datastore.Key) error {
	defer recordLatency(mt.m.batchDeleteLatency, time.Now())
	mt.m.batchDeleteNum.Inc()
	err := mt.b.Delete(ctx, key)
	if err != nil {
		mt.m.batchDeleteErr.Inc()
	}
	return err
}

func (mt *measuredBatch) Commit(ctx context.Context) error {
	defer recordLatency(mt.m.batchCommitLatency, time.Now())
	mt.m.batchCommitNum.Inc()
	err := mt.b.Commit(ctx)
	if err != nil {
		mt.m.batchCommitErr.Inc()
	}
	return err
}

func (m *measure) Close() error {
	if c, ok := m.backend.(io.Closer); ok {
		return c.Close()
	}
	return nil
}
