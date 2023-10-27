package config

import "github.com/libp2p/go-libp2p-core/routing"

type QuorumOptionKey struct{}

const defaultQuorum = 0

// GetQuorum defaults to 0 if no option is found
func GetQuorum(opts *routing.Options) int {
	responsesNeeded, ok := opts.Other[QuorumOptionKey{}].(int)
	if !ok {
		responsesNeeded = defaultQuorum
	}
	return responsesNeeded
}
