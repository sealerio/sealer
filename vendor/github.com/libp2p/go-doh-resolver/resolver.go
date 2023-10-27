package doh

import (
	"context"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"

	madns "github.com/multiformats/go-multiaddr-dns"
)

type Resolver struct {
	mx sync.Mutex
	url string

	// RR cache
	ipCache  map[string]ipAddrEntry
	txtCache map[string]txtEntry
}

type ipAddrEntry struct {
	ips    []net.IPAddr
	expire time.Time
}

type txtEntry struct {
	txt    []string
	expire time.Time
}

func NewResolver(url string) *Resolver {
	if !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}

	return &Resolver{
		url:      url,
		ipCache:  make(map[string]ipAddrEntry),
		txtCache: make(map[string]txtEntry),
	}
}

var _ madns.BasicResolver = (*Resolver)(nil)

func (r *Resolver) LookupIPAddr(ctx context.Context, domain string) (result []net.IPAddr, err error) {
	result, ok := r.getCachedIPAddr(domain)
	if ok {
		return result, nil
	}

	type response struct {
		ips []net.IPAddr
		ttl uint32
		err error
	}

	resch := make(chan response, 2)
	go func() {
		ip4, ttl, err := doRequestA(ctx, r.url, domain)
		resch <- response{ip4, ttl, err}
	}()

	go func() {
		ip6, ttl, err := doRequestAAAA(ctx, r.url, domain)
		resch <- response{ip6, ttl, err}
	}()

	var ttl uint32
	for i := 0; i < 2; i++ {
		r := <-resch
		if r.err != nil {
			return nil, r.err
		}

		result = append(result, r.ips...)
		if ttl == 0 || r.ttl < ttl {
			ttl = r.ttl
		}
	}

	r.cacheIPAddr(domain, result, ttl)
	return result, nil
}

func (r *Resolver) LookupTXT(ctx context.Context, domain string) ([]string, error) {
	result, ok := r.getCachedTXT(domain)
	if ok {
		return result, nil
	}

	result, ttl, err := doRequestTXT(ctx, r.url, domain)
	if err != nil {
		return nil, err
	}

	r.cacheTXT(domain, result, ttl)
	return result, nil
}

func (r *Resolver) getCachedIPAddr(domain string) ([]net.IPAddr, bool) {
	r.mx.Lock()
	defer r.mx.Unlock()

	fqdn := dns.Fqdn(domain)
	entry, ok := r.ipCache[fqdn]
	if !ok {
		return nil, false
	}

	if time.Now().After(entry.expire) {
		delete(r.ipCache, fqdn)
		return nil, false
	}

	return entry.ips, true
}

func (r *Resolver) cacheIPAddr(domain string, ips []net.IPAddr, ttl uint32) {
	if ttl == 0 {
		return
	}

	r.mx.Lock()
	defer r.mx.Unlock()

	fqdn := dns.Fqdn(domain)
	r.ipCache[fqdn] = ipAddrEntry{ips, time.Now().Add(time.Duration(ttl) * time.Second)}
}

func (r *Resolver) getCachedTXT(domain string) ([]string, bool) {
	r.mx.Lock()
	defer r.mx.Unlock()

	fqdn := dns.Fqdn(domain)
	entry, ok := r.txtCache[fqdn]
	if !ok {
		return nil, false
	}

	if time.Now().After(entry.expire) {
		delete(r.txtCache, fqdn)
		return nil, false
	}

	return entry.txt, true
}

func (r *Resolver) cacheTXT(domain string, txt []string, ttl uint32) {
	if ttl == 0 {
		return
	}

	r.mx.Lock()
	defer r.mx.Unlock()

	fqdn := dns.Fqdn(domain)
	r.txtCache[fqdn] = txtEntry{txt, time.Now().Add(time.Duration(ttl) * time.Second)}
}
