package peercube

import "sync"

type Cluster struct {
	Label ID
	Dim   uint64

	mutex *sync.RWMutex
	Vc    map[string]*Peer
	Vs    map[string]*Peer
	Vt    map[string]*Peer
}

func (c *Cluster) copyVc() map[string]*Peer {
	Vc := make(map[string]*Peer)
	for id, peer := range c.Vc {
		Vc[id] = peer
	}
	return Vc
}

func (c *Cluster) copyVs() map[string]*Peer {
	Vs := make(map[string]*Peer)
	for id, peer := range c.Vs {
		Vs[id] = peer
	}
	return Vs
}

func (c *Cluster) copyVt() map[string]*Peer {
	Vt := make(map[string]*Peer)
	for id, peer := range c.Vt {
		Vt[id] = peer
	}
	return Vt
}

func (c *Cluster) coreRandomPeer() *Peer {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	for _, peer := range c.Vc {
		return peer
	}
	return nil
}

func (c *Cluster) tempShortestPrefix() (ID, uint64) {
	check := make(map[uint64]map[string]uint64)
	for _, peer := range c.Vt {
		for i := uint64(0); i < M; i++ {
			prefix := peer.ID[:i]
			check[i][prefix.String()]++
		}
	}
	for _, prefixes := range check {
		for prefix, count := range prefixes {
			if count >= TSplit {
				return IDFromString(prefix), count
			}
		}
	}
	return ID{}, 0
}

func (c *Cluster) isSplit() bool {
	var count0 uint64
	var count1 uint64
	if c.Dim+1 == M {
		return false
	}
	for _, peer := range c.Vc {
		if peer.ID[c.Dim+1] == 1 {
			count1++
		} else {
			count0++
		}
	}
	for _, peer := range c.Vs {
		if peer.ID[c.Dim+1] == 1 {
			count1++
		} else {
			count0++
		}
	}
	return count1 >= TSplit && count0 >= TSplit
}

func (c *Cluster) tempIsSplit() bool {
	return false
}

func (c *Cluster) tempSplit() {
	return
}
func (c *Cluster) lock() {
	if c.mutex == nil {
		c.mutex = &sync.RWMutex{}
	}
}
