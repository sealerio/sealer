package defaults

import (
	"time"
)

const (
	// these requests take at _least_ two minutes at the moment.
	ProvideTimeout  = time.Minute * 3
	ProvSearchDelay = time.Second

	// Number of concurrent workers in decision engine that process requests to the blockstore
	BitswapEngineBlockstoreWorkerCount = 128
	// the total number of simultaneous threads sending outgoing messages
	BitswapTaskWorkerCount = 8
	// how many worker threads to start for decision engine task worker
	BitswapEngineTaskWorkerCount = 8
	// the total amount of bytes that a peer should have outstanding, it is utilized by the decision engine
	BitswapMaxOutstandingBytesPerPeer = 1 << 20
)
