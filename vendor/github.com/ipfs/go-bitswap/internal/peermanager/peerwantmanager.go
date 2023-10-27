package peermanager

import (
	"bytes"
	"fmt"

	cid "github.com/ipfs/go-cid"
	peer "github.com/libp2p/go-libp2p-core/peer"
)

// Gauge can be used to keep track of a metric that increases and decreases
// incrementally. It is used by the peerWantManager to track the number of
// want-blocks that are active (ie sent but no response received)
type Gauge interface {
	Inc()
	Dec()
}

// peerWantManager keeps track of which want-haves and want-blocks have been
// sent to each peer, so that the PeerManager doesn't send duplicates.
type peerWantManager struct {
	// peerWants maps peers to outstanding wants.
	// A peer's wants is the _union_ of the broadcast wants and the wants in
	// this list.
	peerWants map[peer.ID]*peerWant

	// Reverse index of all wants in peerWants.
	wantPeers map[cid.Cid]map[peer.ID]struct{}

	// broadcastWants tracks all the current broadcast wants.
	broadcastWants *cid.Set

	// Keeps track of the number of active want-haves & want-blocks
	wantGauge Gauge
	// Keeps track of the number of active want-blocks
	wantBlockGauge Gauge
}

type peerWant struct {
	wantBlocks *cid.Set
	wantHaves  *cid.Set
	peerQueue  PeerQueue
}

// New creates a new peerWantManager with a Gauge that keeps track of the
// number of active want-blocks (ie sent but no response received)
func newPeerWantManager(wantGauge Gauge, wantBlockGauge Gauge) *peerWantManager {
	return &peerWantManager{
		broadcastWants: cid.NewSet(),
		peerWants:      make(map[peer.ID]*peerWant),
		wantPeers:      make(map[cid.Cid]map[peer.ID]struct{}),
		wantGauge:      wantGauge,
		wantBlockGauge: wantBlockGauge,
	}
}

// addPeer adds a peer whose wants we need to keep track of. It sends the
// current list of broadcast wants to the peer.
func (pwm *peerWantManager) addPeer(peerQueue PeerQueue, p peer.ID) {
	if _, ok := pwm.peerWants[p]; ok {
		return
	}

	pwm.peerWants[p] = &peerWant{
		wantBlocks: cid.NewSet(),
		wantHaves:  cid.NewSet(),
		peerQueue:  peerQueue,
	}

	// Broadcast any live want-haves to the newly connected peer
	if pwm.broadcastWants.Len() > 0 {
		wants := pwm.broadcastWants.Keys()
		peerQueue.AddBroadcastWantHaves(wants)
	}
}

// RemovePeer removes a peer and its associated wants from tracking
func (pwm *peerWantManager) removePeer(p peer.ID) {
	pws, ok := pwm.peerWants[p]
	if !ok {
		return
	}

	// Clean up want-blocks
	_ = pws.wantBlocks.ForEach(func(c cid.Cid) error {
		// Clean up want-blocks from the reverse index
		pwm.reverseIndexRemove(c, p)

		// Decrement the gauges by the number of pending want-blocks to the peer
		peerCounts := pwm.wantPeerCounts(c)
		if peerCounts.wantBlock == 0 {
			pwm.wantBlockGauge.Dec()
		}
		if !peerCounts.wanted() {
			pwm.wantGauge.Dec()
		}

		return nil
	})

	// Clean up want-haves
	_ = pws.wantHaves.ForEach(func(c cid.Cid) error {
		// Clean up want-haves from the reverse index
		pwm.reverseIndexRemove(c, p)

		// Decrement the gauge by the number of pending want-haves to the peer
		peerCounts := pwm.wantPeerCounts(c)
		if !peerCounts.wanted() {
			pwm.wantGauge.Dec()
		}
		return nil
	})

	delete(pwm.peerWants, p)
}

// broadcastWantHaves sends want-haves to any peers that have not yet been sent them.
func (pwm *peerWantManager) broadcastWantHaves(wantHaves []cid.Cid) {
	unsent := make([]cid.Cid, 0, len(wantHaves))
	for _, c := range wantHaves {
		if pwm.broadcastWants.Has(c) {
			// Already a broadcast want, skip it.
			continue
		}
		pwm.broadcastWants.Add(c)
		unsent = append(unsent, c)

		// If no peer has a pending want for the key
		if _, ok := pwm.wantPeers[c]; !ok {
			// Increment the total wants gauge
			pwm.wantGauge.Inc()
		}
	}

	if len(unsent) == 0 {
		return
	}

	// Allocate a single buffer to filter broadcast wants for each peer
	bcstWantsBuffer := make([]cid.Cid, 0, len(unsent))

	// Send broadcast wants to each peer
	for _, pws := range pwm.peerWants {
		peerUnsent := bcstWantsBuffer[:0]
		for _, c := range unsent {
			// If we've already sent a want to this peer, skip them.
			if !pws.wantBlocks.Has(c) && !pws.wantHaves.Has(c) {
				peerUnsent = append(peerUnsent, c)
			}
		}

		if len(peerUnsent) > 0 {
			pws.peerQueue.AddBroadcastWantHaves(peerUnsent)
		}
	}
}

// sendWants only sends the peer the want-blocks and want-haves that have not
// already been sent to it.
func (pwm *peerWantManager) sendWants(p peer.ID, wantBlocks []cid.Cid, wantHaves []cid.Cid) {
	fltWantBlks := make([]cid.Cid, 0, len(wantBlocks))
	fltWantHvs := make([]cid.Cid, 0, len(wantHaves))

	// Get the existing want-blocks and want-haves for the peer
	pws, ok := pwm.peerWants[p]
	if !ok {
		// In practice this should never happen
		log.Errorf("sendWants() called with peer %s but peer not found in peerWantManager", string(p))
		return
	}

	// Iterate over the requested want-blocks
	for _, c := range wantBlocks {
		// If the want-block hasn't been sent to the peer
		if pws.wantBlocks.Has(c) {
			continue
		}

		// Increment the want gauges
		peerCounts := pwm.wantPeerCounts(c)
		if peerCounts.wantBlock == 0 {
			pwm.wantBlockGauge.Inc()
		}
		if !peerCounts.wanted() {
			pwm.wantGauge.Inc()
		}

		// Make sure the CID is no longer recorded as a want-have
		pws.wantHaves.Remove(c)

		// Record that the CID was sent as a want-block
		pws.wantBlocks.Add(c)

		// Add the CID to the results
		fltWantBlks = append(fltWantBlks, c)

		// Update the reverse index
		pwm.reverseIndexAdd(c, p)
	}

	// Iterate over the requested want-haves
	for _, c := range wantHaves {
		// If we've already broadcasted this want, don't bother with a
		// want-have.
		if pwm.broadcastWants.Has(c) {
			continue
		}

		// If the CID has not been sent as a want-block or want-have
		if !pws.wantBlocks.Has(c) && !pws.wantHaves.Has(c) {
			// Increment the total wants gauge
			peerCounts := pwm.wantPeerCounts(c)
			if !peerCounts.wanted() {
				pwm.wantGauge.Inc()
			}

			// Record that the CID was sent as a want-have
			pws.wantHaves.Add(c)

			// Add the CID to the results
			fltWantHvs = append(fltWantHvs, c)

			// Update the reverse index
			pwm.reverseIndexAdd(c, p)
		}
	}

	// Send the want-blocks and want-haves to the peer
	pws.peerQueue.AddWants(fltWantBlks, fltWantHvs)
}

// sendCancels sends a cancel to each peer to which a corresponding want was
// sent
func (pwm *peerWantManager) sendCancels(cancelKs []cid.Cid) {
	if len(cancelKs) == 0 {
		return
	}

	// Record how many peers have a pending want-block and want-have for each
	// key to be cancelled
	peerCounts := make(map[cid.Cid]wantPeerCnts, len(cancelKs))
	for _, c := range cancelKs {
		peerCounts[c] = pwm.wantPeerCounts(c)
	}

	// Create a buffer to use for filtering cancels per peer, with the
	// broadcast wants at the front of the buffer (broadcast wants are sent to
	// all peers)
	broadcastCancels := make([]cid.Cid, 0, len(cancelKs))
	for _, c := range cancelKs {
		if pwm.broadcastWants.Has(c) {
			broadcastCancels = append(broadcastCancels, c)
		}
	}

	// Send cancels to a particular peer
	send := func(p peer.ID, pws *peerWant) {
		// Start from the broadcast cancels
		toCancel := broadcastCancels

		// For each key to be cancelled
		for _, c := range cancelKs {
			// Check if a want was sent for the key
			if !pws.wantBlocks.Has(c) && !pws.wantHaves.Has(c) {
				continue
			}

			// Unconditionally remove from the want lists.
			pws.wantBlocks.Remove(c)
			pws.wantHaves.Remove(c)

			// If it's a broadcast want, we've already added it to
			// the peer cancels.
			if !pwm.broadcastWants.Has(c) {
				toCancel = append(toCancel, c)
			}
		}

		// Send cancels to the peer
		if len(toCancel) > 0 {
			pws.peerQueue.AddCancels(toCancel)
		}
	}

	if len(broadcastCancels) > 0 {
		// If a broadcast want is being cancelled, send the cancel to all
		// peers
		for p, pws := range pwm.peerWants {
			send(p, pws)
		}
	} else {
		// Only send cancels to peers that received a corresponding want
		cancelPeers := make(map[peer.ID]struct{}, len(pwm.wantPeers[cancelKs[0]]))
		for _, c := range cancelKs {
			for p := range pwm.wantPeers[c] {
				cancelPeers[p] = struct{}{}
			}
		}
		for p := range cancelPeers {
			pws, ok := pwm.peerWants[p]
			if !ok {
				// Should never happen but check just in case
				log.Errorf("sendCancels - peerWantManager index missing peer %s", p)
				continue
			}

			send(p, pws)
		}
	}

	// Decrement the wants gauges
	for _, c := range cancelKs {
		peerCnts := peerCounts[c]

		// If there were any peers that had a pending want-block for the key
		if peerCnts.wantBlock > 0 {
			// Decrement the want-block gauge
			pwm.wantBlockGauge.Dec()
		}

		// If there was a peer that had a pending want or it was a broadcast want
		if peerCnts.wanted() {
			// Decrement the total wants gauge
			pwm.wantGauge.Dec()
		}
	}

	// Remove cancelled broadcast wants
	for _, c := range broadcastCancels {
		pwm.broadcastWants.Remove(c)
	}

	// Batch-remove the reverse-index. There's no need to clear this index
	// peer-by-peer.
	for _, c := range cancelKs {
		delete(pwm.wantPeers, c)
	}
}

// wantPeerCnts stores the number of peers that have pending wants for a CID
type wantPeerCnts struct {
	// number of peers that have a pending want-block for the CID
	wantBlock int
	// number of peers that have a pending want-have for the CID
	wantHave int
	// whether the CID is a broadcast want
	isBroadcast bool
}

// wanted returns true if any peer wants the CID or it's a broadcast want
func (pwm *wantPeerCnts) wanted() bool {
	return pwm.wantBlock > 0 || pwm.wantHave > 0 || pwm.isBroadcast
}

// wantPeerCounts counts how many peers have a pending want-block and want-have
// for the given CID
func (pwm *peerWantManager) wantPeerCounts(c cid.Cid) wantPeerCnts {
	blockCount := 0
	haveCount := 0
	for p := range pwm.wantPeers[c] {
		pws, ok := pwm.peerWants[p]
		if !ok {
			log.Errorf("reverse index has extra peer %s for key %s in peerWantManager", string(p), c)
			continue
		}

		if pws.wantBlocks.Has(c) {
			blockCount++
		} else if pws.wantHaves.Has(c) {
			haveCount++
		}
	}

	return wantPeerCnts{blockCount, haveCount, pwm.broadcastWants.Has(c)}
}

// Add the peer to the list of peers that have sent a want with the cid
func (pwm *peerWantManager) reverseIndexAdd(c cid.Cid, p peer.ID) bool {
	peers, ok := pwm.wantPeers[c]
	if !ok {
		peers = make(map[peer.ID]struct{}, 10)
		pwm.wantPeers[c] = peers
	}
	peers[p] = struct{}{}
	return !ok
}

// Remove the peer from the list of peers that have sent a want with the cid
func (pwm *peerWantManager) reverseIndexRemove(c cid.Cid, p peer.ID) {
	if peers, ok := pwm.wantPeers[c]; ok {
		delete(peers, p)
		if len(peers) == 0 {
			delete(pwm.wantPeers, c)
		}
	}
}

// GetWantBlocks returns the set of all want-blocks sent to all peers
func (pwm *peerWantManager) getWantBlocks() []cid.Cid {
	res := cid.NewSet()

	// Iterate over all known peers
	for _, pws := range pwm.peerWants {
		// Iterate over all want-blocks
		_ = pws.wantBlocks.ForEach(func(c cid.Cid) error {
			// Add the CID to the results
			res.Add(c)
			return nil
		})
	}

	return res.Keys()
}

// GetWantHaves returns the set of all want-haves sent to all peers
func (pwm *peerWantManager) getWantHaves() []cid.Cid {
	res := cid.NewSet()

	// Iterate over all peers with active wants.
	for _, pws := range pwm.peerWants {
		// Iterate over all want-haves
		_ = pws.wantHaves.ForEach(func(c cid.Cid) error {
			// Add the CID to the results
			res.Add(c)
			return nil
		})
	}
	_ = pwm.broadcastWants.ForEach(func(c cid.Cid) error {
		res.Add(c)
		return nil
	})

	return res.Keys()
}

// GetWants returns the set of all wants (both want-blocks and want-haves).
func (pwm *peerWantManager) getWants() []cid.Cid {
	res := pwm.broadcastWants.Keys()

	// Iterate over all targeted wants, removing ones that are also in the
	// broadcast list.
	for c := range pwm.wantPeers {
		if pwm.broadcastWants.Has(c) {
			continue
		}
		res = append(res, c)
	}

	return res
}

func (pwm *peerWantManager) String() string {
	var b bytes.Buffer
	for p, ws := range pwm.peerWants {
		b.WriteString(fmt.Sprintf("Peer %s: %d want-have / %d want-block:\n", p, ws.wantHaves.Len(), ws.wantBlocks.Len()))
		for _, c := range ws.wantHaves.Keys() {
			b.WriteString(fmt.Sprintf("  want-have  %s\n", c))
		}
		for _, c := range ws.wantBlocks.Keys() {
			b.WriteString(fmt.Sprintf("  want-block %s\n", c))
		}
	}
	return b.String()
}
