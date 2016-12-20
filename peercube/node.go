package peercube

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"sync"
)

type PeerType byte

type Peer struct {
	//General info
	ID   ID
	Type PeerType
	N    int

	//For Serving

	pt             map[string]*Cluster
	rt             []*Cluster
	cluster        *Cluster
	data           KVStore
	joinlock       *sync.Mutex
	splitlock      *sync.RWMutex
	mutex          *sync.RWMutex
	eventreg       EventRegistry
	eventq         chan Event
	msgq           chan Message
	quit           bool
	joinackhandler *JoinAckHandler
}

func NewPeer() *Peer {
	eventq := make(chan Event, QueueSize)
	p := &Peer{
		N:              len(peerregistry.data),
		ID:             RandomID(M),
		splitlock:      &sync.RWMutex{},
		mutex:          &sync.RWMutex{},
		eventq:         eventq,
		eventreg:       newmsgcount(),
		pt:             make(map[string]*Cluster),
		rt:             make([]*Cluster, 0, M),
		msgq:           make(chan Message, QueueSize),
		joinlock:       &sync.Mutex{},
		joinackhandler: NewJoinAckHandler(int((SMin-1)/3+1), eventq),
		quit:           false,
		Type:           UNDEF,
	}
	p.cluster = &Cluster{
		Dim:   0,
		Label: make(ID, 0),
		Vc: map[string]*Peer{
			p.ID.String(): p,
		},
		Vs:    make(map[string]*Peer),
		Vt:    make(map[string]*Peer),
		mutex: &sync.RWMutex{},
	}
	peerregistry.Set(p.ID.String(), p)

	return p
}

func (p *Peer) findClosestCluster(id ID) *Cluster {
	if p.Type != CORE {
		return p.cluster
	}
	if p.cluster.Dim == 0 || p.cluster.Label.prefix(id) {
		return p.cluster
	} else {
		//fmt.Println(p.cluster.Dim, p.cluster.Label.String(), len(p.rt), p.ID.String(), p.Type)
		c := p.rt[0]
		for i := uint64(0); i < p.cluster.Dim; i++ {
			if distance(id, p.rt[i].Label).lt(distance(id, c.Label)) {
				c = p.rt[i]
			}
		}
		return c
	}
	return nil
}

func (p *Peer) send(m Message) error {
	var mod bytes.Buffer
	enc := gob.NewEncoder(&mod)
	err := enc.Encode(&m)
	if err != nil {
		fmt.Println(err)
	}
	dec := gob.NewDecoder(&mod)
	var copy Message
	err = dec.Decode(&copy)
	if err != nil {
		fmt.Println(err)
	}
	//fmt.Println(copy.getEvent(), copy.Sender())

	peerregistry.Get(p.ID.String()).msgq <- copy
	return nil
}

func (p *Peer) serve() {
	for i := uint64(0); i < WorkerCount; i++ {
		go func() {
			for !p.quit {
				select {
				case msg := <-p.msgq:
					if msg == nil {
						fmt.Println("NIL MESSAGE")
					}
					switch msg.Type() {
					case JOINACK:
						p.joinackhandler.Trigger(msg)
					case PUT, LOOKUP:
						p.eventreg.Register(msg.ResponseHash(), NewFanInOutHandler(int((SMin-1)/3+1), p.eventq))
						p.eventreg.Register(msg.Hash(), NewFanInHandler(int((SMin-1)/3+1), msg.getEvent(), p.eventq))
						p.eventreg.Check(msg)
					case JOIN:
						if msg.Sender().Type == UNDEF {
							p.eventq <- msg.getEvent()
							continue
						}
						//fmt.Println(msg.Hash(), p.n, p.cluster.Label)
						p.eventreg.Register(
							msg.Hash(),
							NewFanInHandler(int((SMin-1)/3), msg.getEvent(), p.eventq),
						)
						p.eventreg.Check(msg)
					case JOINSPARE, JOINTEMP:
						p.eventreg.Register(
							msg.Hash(),
							NewFanInHandler(int((SMin-1)/3+1), msg.getEvent(), p.eventq),
						)
						p.eventreg.Check(msg)
					case SPLIT, DISCONNECT, CONNECT:
						p.eventreg.Register(msg.Hash(), NewFanInHandler(int((SMin-1)/3+1), msg.getEvent(), p.eventq))
						p.eventreg.Check(msg)
					}
				case event := <-p.eventq:
					if event.Safe() {
						p.splitlock.RLock()
						event.fire(p)
						p.splitlock.RUnlock()
					} else {
						p.splitlock.Lock()
						event.fire(p)
						p.splitlock.Unlock()
					}
				}

			}
		}()
	}
}

func (p *Peer) joinByID(id ID) error {
	seed := peerregistry.Get(id.String())
	if seed == nil {
		return errors.New("No such peer" + id.String())
	}
	var msg Message
	p.joinlock.Lock()
	seed.cluster.mutex.RLock()
	for _, peer := range seed.cluster.Vc {
		msg = NewJoin(p, p)
		peer.send(msg)
	}
	seed.cluster.mutex.RUnlock()
	go p.serve()
	p.joinlock.Lock()
	p.joinlock.Unlock()
	return nil
}

func (p *Peer) split() ([]*Cluster, map[string]*Cluster) {
	//Not Safe for concurrent calls
	label := make([]ID, 2)
	Vc := make([]map[string]*Peer, 2)
	Vs := make([]map[string]*Peer, 2)
	for i := 0; i <= 1; i++ {
		label[i] = make(ID, len(p.cluster.Label)+1, M)
		copy(label[i], p.cluster.Label)
		label[i][len(p.cluster.Label)] = byte(i)

		Vc[i] = make(map[string]*Peer)
		Vs[i] = make(map[string]*Peer)
		for id, peer := range p.cluster.Vc {
			if label[i].prefix(peer.ID) {
				Vc[i][id] = peer
			}
		}
		for id, peer := range p.cluster.Vs {
			if label[i].prefix(peer.ID) {
				Vs[i][id] = peer
			}
		}
	}
	oVs := make([]map[string]*Peer, 2)
	for i := 0; i <= 1; i++ {
		oVs[i] = make(map[string]*Peer)
		for id, peer := range Vs[i] {
			oVs[i][id] = peer
		}
	}
	for i := 0; i <= 1; i++ {
		for len(Vc[i]) < int(SMin) {
			var min *Peer
			for _, peer := range Vs[i] {
				if min == nil || min.ID.gt(peer.ID) {
					min = peer
				}
			}
			strk := min.ID.String()
			delete(Vs[i], strk)
			Vc[i][strk] = min

		}
	}
	p.cluster.Dim++

	var lrt, ort []*Cluster
	var lpt, opt map[string]*Cluster

	for _, rc := range p.rt {
		for _, peer := range rc.Vc {
			peer.send(NewDisconnectInform(p, p.cluster))
		}
	}

	for i := 0; i <= 1; i++ {
		cluster := &Cluster{
			Label: label[i],
			Dim:   p.cluster.Dim,
			Vc:    Vc[i],
			Vs:    Vs[i],
		}
		ocluster := &Cluster{
			Label: label[i^1],
			Dim:   p.cluster.Dim,
			Vc:    Vc[i^1],
			Vs:    Vs[i^1],
		}
		rt := make([]*Cluster, len(p.rt)+1, M)
		copy(rt, p.rt)
		rt[len(p.rt)] = ocluster
		pt := make(map[string]*Cluster)
		for l, c := range p.pt {
			pt[l] = c
		}
		pt[ocluster.Label.String()] = ocluster
		for j, rc := range rt {
			ideal := make(ID, cluster.Dim)
			copy(ideal, label[i])
			ideal[j] = ideal[j] ^ 1

			old := rc
			for _, pc := range pt {
				if distance(rc.Label, ideal).gte(distance(pc.Label, ideal)) && rc.Dim < pc.Dim {
					rt[j] = pc
				}
			}
			fmt.Println("Disconnecting from", old.Label.String(), " to ", rc.Label.String())
			for _, peer := range rc.Vc {
				peer.send(NewDisconnectInform(p, cluster))
			}
		}
		for id, peer := range oVs[i] {
			if _, found := Vc[i][id]; found {
				peer.send(NewSplit(p, &State{
					Cluster: &Cluster{
						Label: label[i],
						Dim:   p.cluster.Dim,
						Vc:    Vc[i],
						Vs:    Vs[i],
					},
					Rt:   rt,
					Pt:   pt,
					Type: CORE,
				}))
			} else {
				peer.send(NewSplit(p, &State{
					Cluster: &Cluster{
						Label: label[i],
						Dim:   cluster.Dim,
						Vc:    Vc[i],
					},
					Type: SPARE,
				}))
			}
		}
		for _, c := range rt {
			for _, peer := range c.Vc {
				peer.send(NewConnectInform(p, cluster))
			}
		}
		if label[i].prefix(p.ID) {
			lpt = pt
			lrt = rt
		} else {
			opt = pt
			ort = rt
		}
	}
	p.rt = lrt
	p.pt = lpt
	if label[0].prefix(p.ID) {
		p.cluster.Label = label[0]
		p.cluster.Vc = Vc[0]
		p.cluster.Vs = Vs[0]
	} else {
		p.cluster.Label = label[1]
		p.cluster.Vc = Vc[1]
		p.cluster.Vs = Vs[1]
	}

	return ort, opt
}

/*
func (p *Peer) lookup(key *ID, q *Peer) (*Resp, error) {
	c := p.findClosestCluster(*key)
	timeMult := time.Duration(hamming_distance(c.Label, (*key)[:len(c.Label)])) + 2

	if p.cluster.Label.eq(c.Label) {

		//Ensure following operation not done twice
		if p.respquora.Register(key, (SMin-1)/3+1) {
			//If not done before, send to all of coreset
			//Broadcast, not implemented exactly yet, fudging it for now

			for _, peer := range p.cluster.Vc {

				go func(peer *Peer) {
					var resp *Resp
					var done chan struct{}
					go func() {
						peer.lookup(key, q)
						done <- struct{}{}
					}()

					select {
					case <-done:
						p.respquora.Check(key, resp)
						return
					case <-p.respquora.Done(key):
						return
					case <-time.After(p.timeout * timeMult):
						return
					}
				}(peer)
			}
			return p.respquora.Wait(key), nil
		}

		return &Resp{
			Key:   *key,
			Label: p.cluster.Label,
			Value: p.data.Get(key.String()),
		}, nil
	}
	if !p.respquora.Register(key, (SMin-1)/3+1) {
		for _, peer := range p.cluster.Vc {
			go func(peer *Peer, key *ID) {
				var resp *Resp
				var done chan struct{}
				go func() {
					peer.lookup(key, q)
					done <- struct{}{}
				}()

				select {
				case <-done:
					p.respquora.Check(key, resp)
					return
				case <-p.respquora.Done(key):
					return
				case <-time.After(p.timeout * timeMult):
					return
				}
			}(peer, key)
		}
	}
	return p.respquora.Wait(key), nil
}
*/
