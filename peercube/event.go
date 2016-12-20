package peercube

import (
	"encoding/gob"
	"fmt"
	"sync"
)

func init() {
	gob.Register(&LookupEvent{})
	gob.Register(&JoinEvent{})
	gob.Register(&JoinSpareEvent{})
	gob.Register(&JoinTempEvent{})
	gob.Register(&JoinAckEvent{})
	gob.Register(&SplitEvent{})
	gob.Register(&ConnectInformEvent{})
	gob.Register(&DisconnectInformEvent{})
}

type Event interface {
	fire(p *Peer) error
	Safe() bool
	Type() MsgType
}

type LookupEvent struct {
	Prev   *Peer
	Origin *Peer
	Key    ID
}

func (e *LookupEvent) Safe() bool {
	return true
}

func (e *LookupEvent) Type() MsgType {
	return LOOKUP
}

func (e *LookupEvent) fire(p *Peer) error {
	return nil
}

type JoinEvent struct {
	Origin *Peer
}

func (e *JoinEvent) Safe() bool {
	return false
}

func (e *JoinEvent) Type() MsgType {
	return JOIN
}
func (e *JoinEvent) fire(p *Peer) error {

	p.mutex.RLock()
	defer p.mutex.RUnlock()
	c := p.findClosestCluster(e.Origin.ID)
	if p.cluster.Label.eq(c.Label) {
		if p.cluster.Label.prefix(e.Origin.ID) {
			for _, peer := range p.cluster.Vc {
				if peer.ID.eq(p.ID) {
					continue
				}
				peer.send(NewJoinSpare(p, e.Origin))
			}
			return nil
		}
		for _, peer := range p.cluster.Vc {
			if peer.ID.eq(p.ID) {
				continue
			}
			peer.send(NewJoinTemp(p, e.Origin))
		}
		return nil
	}
	for _, peer := range c.Vc {
		peer.send(NewJoin(p, e.Origin))
	}
	//fmt.Println(e.origin.ID.String(), "forwarded to", c.Label.String())
	return nil
}

type JoinSpareEvent struct {
	Origin *Peer
}

func (e *JoinSpareEvent) Safe() bool {
	return false
}

func (e *JoinSpareEvent) Type() MsgType {
	return JOINSPARE
}

func (e *JoinSpareEvent) fire(p *Peer) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if _, found := p.cluster.Vs[e.Origin.ID.String()]; found {
		return nil
	}
	p.cluster.Vs[e.Origin.ID.String()] = e.Origin

	var ort []*Cluster
	var opt map[string]*Cluster
	if p.cluster.isSplit() {
		//fmt.Println("SPLITTING", p.cluster.Label.String(), "into", append(p.cluster.Label, 1).String(), "and", append(p.cluster.Label, 0).String())
		ort, opt = p.split()
	}

	c := p.findClosestCluster(e.Origin.ID)
	state := &State{
		Cluster: &Cluster{
			Label: c.Label,
			Dim:   c.Dim,
			Vc:    c.Vc,
			mutex: &sync.RWMutex{},
		},
		Type: SPARE,
	}
	if _, found := c.Vc[e.Origin.ID.String()]; found {
		state.Cluster.Vs = c.Vs
		state.Cluster.Vt = c.Vt
		if c.Label.eq(p.cluster.Label) {
			state.Pt = make(map[string]*Cluster)
			for cid, c := range p.pt {
				state.Pt[cid] = c
			}
			state.Rt = make([]*Cluster, len(p.rt), M)
			copy(state.Rt, p.rt)
		} else {
			state.Rt = ort
			state.Pt = opt
		}
		state.Type = CORE
	}
	e.Origin.send(NewJoinAck(p, state))
	return nil
}

type JoinTempEvent struct {
	Origin *Peer
}

func (e *JoinTempEvent) Safe() bool {
	return false
}

func (e *JoinTempEvent) Type() MsgType {
	return JOINTEMP
}

func (e *JoinTempEvent) fire(p *Peer) error {
	return nil
}

type JoinAckEvent struct {
	State *State
}

func (e *JoinAckEvent) Safe() bool {
	return false
}

func (e *JoinAckEvent) Type() MsgType {
	return JOINACK
}

func (e *JoinAckEvent) fire(p *Peer) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.cluster = e.State.Cluster
	p.cluster.mutex = &sync.RWMutex{}
	p.Type = e.State.Type
	if e.State.Type == CORE {
		p.rt = e.State.Rt
		p.pt = e.State.Pt
	}
	p.joinlock.Unlock()
	return nil
}

type SplitEvent struct {
	State *State
}

func (e *SplitEvent) Safe() bool {
	return false
}

func (e *SplitEvent) Type() MsgType {
	return SPLIT
}

func (e *SplitEvent) fire(p *Peer) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.Type = e.State.Type
	p.cluster = e.State.Cluster
	p.rt = e.State.Rt
	if e.State.Type == CORE {
		p.rt = e.State.Rt
		p.pt = e.State.Pt
	}

	// /fmt.Println("Splat", p.ID.String(), e.State.Cluster.Label.String(), e.State.Type, len(p.rt), len(e.State.Rt))
	return nil
}

type ConnectInformEvent struct {
	Cluster *Cluster
}

func (e *ConnectInformEvent) Safe() bool {
	return false
}

func (e *ConnectInformEvent) Type() MsgType {
	return CONNECT
}

func (e *ConnectInformEvent) fire(p *Peer) error {
	//fmt.Println("FIRE informing", p.ID.String(), " at ", p.cluster.Label.String(), " of ", e.cluster.Label)
	p.mutex.Lock()
	defer p.mutex.Unlock()
	fmt.Printf("%3s | %3s\n", p.cluster.Label.String(), e.Cluster.Label.String())
	p.pt[e.Cluster.Label.String()] = e.Cluster
	for j, rc := range p.rt {
		ideal := make(ID, M)
		copy(ideal, p.cluster.Label)
		ideal[j] = ideal[j] ^ 1
		if distance(rc.Label, ideal).gte(distance(e.Cluster.Label, ideal)) && rc.Dim < e.Cluster.Dim {
			for _, peer := range rc.Vc {
				peer.send(NewDisconnectInform(p, p.cluster))
			}
			p.rt[j] = e.Cluster
		}
	}
	return nil
}

type DisconnectInformEvent struct {
	Cluster *Cluster
}

func (e *DisconnectInformEvent) Safe() bool {
	return false
}

func (e *DisconnectInformEvent) Type() MsgType {
	return DISCONNECT
}

func (e *DisconnectInformEvent) fire(p *Peer) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	delete(p.pt, e.Cluster.Label.String())
	return nil
}
