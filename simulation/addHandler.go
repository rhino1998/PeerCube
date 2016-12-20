package simulation

import "sync"

type PeerHandler struct {
	mutex     sync.Mutex
	peers     map[string]Peer
	freq      map[string]int
	threshold int
}

func NewPeerHandler(t int) *PeerHandler {
	return &PeerHandler{
		threshold: t,
		freq:      make(map[string]int),
		peers:     make(map[string]Peer),
	}
}

func (ph *PeerHandler) Add(p Peer) bool {

	idstr := p.ID().String()

	ph.mutex.Lock()
	defer ph.mutex.Unlock()

	ph.peers[idstr] = p
	ph.freq[idstr]++

	if ph.freq[idstr] > ph.threshold {
		delete(ph.peers, idstr)
		delete(ph.freq, idstr)
		return true
	}

	return false
}
