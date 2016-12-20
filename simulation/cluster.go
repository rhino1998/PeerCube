package simulation

import (
	"fmt"
	"sync"
	"sync/atomic"
)

type Cluster struct {
	labelMutex sync.RWMutex
	label      ID

	merge uint64

	mutex      sync.RWMutex
	vc         map[string]Peer
	vs         map[string]Peer
	vt         map[string]Peer
	rt         []*Cluster
	addHandler *PeerHandler
}

func NewCluster() *Cluster {
	c := &Cluster{
		label:      make(ID, 0, M),
		vc:         make(map[string]Peer),
		vs:         make(map[string]Peer),
		vt:         make(map[string]Peer),
		rt:         make([]*Cluster, 0, M),
		addHandler: NewPeerHandler(SafteyValue),
	}
	ClusterRegistry.Register(c)
	return c
}

func SplitCluster(label ID, vc, vs map[string]Peer, rt []*Cluster) *Cluster {
	c := &Cluster{
		label:      label,
		vc:         vc,
		vs:         vs,
		vt:         make(map[string]Peer),
		rt:         rt,
		addHandler: NewPeerHandler(SafteyValue),
	}
	for _, p := range c.vc {
		p.SetCluster(c)
		p.SetType(CORE)

	}
	for _, p := range c.vs {
		p.SetCluster(c)
		p.SetType(SPARE)

	}
	fmt.Println(c.label.String(), len(c.rt))
	return c
}

func (c *Cluster) setLabel(label ID) {
	c.labelMutex.Lock()
	defer c.labelMutex.Unlock()
	ClusterRegistry.Update(label, c)
	c.label = label
}

func (c *Cluster) Label() ID {
	c.labelMutex.RLock()
	defer c.labelMutex.RUnlock()
	return c.label
}

func (c *Cluster) Dim() int {
	c.labelMutex.RLock()
	defer c.labelMutex.RUnlock()
	return len(c.label)
}

func (c *Cluster) addVc(p Peer) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.vc[p.ID().String()] = p
	p.SetCluster(c)
	p.SetType(CORE)
}

func (c *Cluster) replaceVC(p Peer) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	var nominee Peer
	for id, sp := range c.vs {
		delete(c.vs, id)
		nominee = sp
		break
	}
	delete(c.vc, p.ID().String())
	if nominee == nil {
		if atomic.SwapUint64(&c.merge, 1) == 1 || len(c.label) == 0 {
			return
		}
		fmt.Println(c.label.String())
		fmt.Println(len(c.label))
		fmt.Println(len(c.rt))
		Merge(c, c.rt[len(c.label)-1]) //MORE OF THEM, FIX SOON
		fmt.Println("NEED TO MERGE")
		return
	}
	c.vc[nominee.ID().String()] = nominee
	nominee.replaceData(p.dataStore().Copy())
	nominee.SetType(CORE)
	defer ClusterRegistry.FixAllRT()
}

func (c *Cluster) delVc(p Peer) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	delete(c.vc, p.ID().String())
}

func (c *Cluster) Add(p Peer, pt PeerType) {
	// /fmt.Println(p.ID().String(), pt, c.Label().String())
	c.mutex.RLock()
	if _, found := c.vs[p.ID().String()]; found {
		c.mutex.RUnlock()
		return
	}
	c.mutex.RUnlock()

	c.mutex.Lock()
	defer c.mutex.Unlock()

	if !c.addHandler.Add(p) {
		return
	}

	defer ClusterRegistry.FixAllRT()

	if !c.Label().prefix(p.ID()) {
		return
	}

	c.vs[p.ID().String()] = p
	//fmt.Println("added", p.ID().String())
	p.SetCluster(c)
	p.SetType(pt)

	if c.isSplit() {
		c.Split()
	}

	p.ready()
}

func (c *Cluster) isSplit() bool {
	var count0 int
	var count1 int
	if len(c.label)+1 == M {
		return false
	}
	for _, peer := range c.vc {
		if peer.ID()[len(c.label)] {
			count1++
		} else {
			count0++
		}
	}
	for _, peer := range c.vs {
		if peer.ID()[len(c.label)] {
			count1++
		} else {
			count0++
		}
	}
	return count1 >= TSplit && count0 >= TSplit
}

func (c *Cluster) Size() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return len(c.vs) + len(c.vc)
}

func (c *Cluster) GetVc() map[string]Peer {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.vc
}

func (c *Cluster) InVs(p Peer) bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	_, found := c.vs[p.ID().String()]
	return found
}

func (c *Cluster) GetVs() map[string]Peer {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.vs
}

func (c *Cluster) SendToVc(msg Message) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	for _, p := range c.vc {
		p.send(msg)
	}
}

func (c *Cluster) getClosestCluster(target ID) *Cluster {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	if len(c.label) == 0 || c.label.prefix(target) {
		return c
	} else {
		//fmt.Println(p.cluster.Dim, p.cluster.Label.String(), len(p.rt), p.ID.String(), p.Type)
		cluster := c.rt[0]
		for i := 0; i < len(c.label); i++ {
			if distance(target, c.rt[i].label).lt(distance(target, cluster.Label())) {
				cluster = c.rt[i]
			}
		}
		return cluster
	}
}

func (c *Cluster) Split() {

	label := make([]ID, 2)
	vc := make([]map[string]Peer, 2)
	vs := make([]map[string]Peer, 2)

	copyPeer := make([]Peer, 2)
	for i := 0; i <= 1; i++ {
		label[i] = make(ID, c.Dim()+1, M)
		copy(label[i], c.Label())
		label[i][len(c.label)] = i == 1

		vc[i] = make(map[string]Peer)
		vs[i] = make(map[string]Peer)
		for id, peer := range c.vc {
			if label[i].prefix(peer.ID()) {
				vc[i][id] = peer
			}
		}
		for id, peer := range c.vs {
			if label[i].prefix(peer.ID()) {
				vs[i][id] = peer
			}
		}
	}
	for i := 0; i <= 1; i++ {
		for len(vc[i]) < int(SMin) {
			var min Peer
			for _, peer := range vs[i] {
				if min == nil || min.ID().gt(peer.ID()) {
					min = peer
				}
			}
			strk := min.ID().String()
			delete(vs[i], strk)
			vc[i][strk] = min

		}
	}
	for i := 0; i <= 1; i++ {
		for _, peer := range vc[i] {
			copyPeer[i] = peer
			break
		}
	}

	for i, peer := range copyPeer {
		peer.dataStore().Filter(label[i])
	}

	for i := 0; i <= 1; i++ {
		for _, peer := range vc[i] {
			peer.replaceData(copyPeer[i].dataStore().Copy())
		}
	}

	rt := make([]*Cluster, len(c.label), M)
	copy(rt, c.rt)
	rt = append(rt, c)

	oc := SplitCluster(label[1], vc[1], vs[1], rt)
	ClusterRegistry.Register(oc)
	c.setLabel(label[0])
	c.rt = append(c.rt, oc)

	c.vc = vc[0]
	c.vs = vs[0]

}
