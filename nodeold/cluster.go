package node

import "sync"

type Cluster struct {
	Label  ID
	Dim    uint64
	M      uint64
	SMax   uint64
	SMin   uint64
	TSplit uint64

	sync.RWMutex
	CoreSet  map[string]*Peer
	SpareSet map[string]*Peer
	TempSet  map[string]*Peer
}

func (c *Cluster) coreRandomPeers(n uint64) []*Peer {
	peers := make([]*Peer, n)
	c.RLock()

	i := uint64(0)
	for _, peer := range c.CoreSet {
		if i == n {
			return peers
		}
		peers[i] = peer
		i++
	}
	c.RUnlock()
	return peers
}

func (c *Cluster) addSpare(p *Peer) {
	c.Lock()
	defer c.Unlock()
	c.SpareSet[p.ID.String()] = p
}

func (c *Cluster) coreRandomPeer() *Peer {
	c.RLock()
	defer c.RUnlock()

	for _, peer := range c.CoreSet {
		return peer
	}
	return nil
}

func (c *Cluster) tempShortestPrefix() (ID, uint64) {
	check := make(map[uint64]map[string]uint64)
	for _, peer := range c.TempSet {
		for i := uint64(0); i < c.M; i++ {
			prefix := peer.ID[:i]
			check[i][prefix.String()]++
		}
	}
	for _, prefixes := range check {
		for prefix, count := range prefixes {
			if count >= c.TSplit {
				return IDFromString(prefix), count
			}
		}
	}
	return ID{}, 0
}

func (c *Cluster) isSplit() bool {
	var count0 uint64
	var count1 uint64
	if c.Dim+1 == c.M {
		return false
	}
	for _, peer := range c.CoreSet {
		if peer.ID[c.Dim+1] {
			count1++
		} else {
			count0++
		}
	}
	for _, peer := range c.SpareSet {
		if peer.ID[c.Dim+1] {
			count1++
		} else {
			count0++
		}
	}
	return count1 >= c.TSplit && count0 >= c.TSplit
}
