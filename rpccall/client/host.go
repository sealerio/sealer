package client

import (
	"strings"
)

type host string

func (h host) ip() string {
	ipv4 := h.String()
	i := strings.IndexRune(h.String(), ':')
	if i >= 0 {
		ipv4 = ipv4[:i]
	}
	return ipv4
}

func (h host) String() string {
	return string(h)
}
