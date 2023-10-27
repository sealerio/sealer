package routinghelpers

import (
	"context"
)

// TODO: Consider moving this to the routing package?

// Bootstrap is an interface that should be implemented by any routers wishing
// to be bootstrapped.
type Bootstrap interface {
	// Bootstrap bootstraps the router.
	Bootstrap(ctx context.Context) error
}
