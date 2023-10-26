package filter

import "github.com/multiformats/go-multiaddr"

// Deprecated. Use "github.com/multiformats/go-multiaddr".Action instead.
type Action = multiaddr.Action

const (
	// Deprecated. Use "github.com/multiformats/go-multiaddr".ActionNone instead.
	ActionNone = multiaddr.ActionNone
	// Deprecated. Use "github.com/multiformats/go-multiaddr".ActionAccept instead.
	ActionAccept = multiaddr.ActionAccept
	// Deprecated. Use "github.com/multiformats/go-multiaddr".ActionDeny instead.
	ActionDeny = multiaddr.ActionDeny
)

// Deprecated. Use "github.com/multiformats/go-multiaddr".Filters instead.
type Filters = multiaddr.Filters

// Deprecated. Use "github.com/multiformats/go-multiaddr".NewFilters instead.
func NewFilters() *multiaddr.Filters {
	return multiaddr.NewFilters()
}
