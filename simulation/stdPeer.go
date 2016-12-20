package simulation

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

type stdPeer struct {
	split    sync.RWMutex
	mutex    sync.RWMutex
	peerType PeerType
	id       ID
	cluster  *Cluster
	eventreg EventRegistry
	eventq   chan Event

	data DataStore

	msgq chan Message
	init chan struct{}
	quit chan struct{}
	dead bool

	putWaitMutex         sync.RWMutex
	putWaitDataRegistry  map[string]chan string
	putWaitErrorRegistry map[string]chan error

	putReqMutex    sync.RWMutex
	putReqRegistry map[string]map[string]Peer

	lookupWaitMutex         sync.RWMutex
	lookupWaitDataRegistry  map[string]chan string
	lookupWaitErrorRegistry map[string]chan error

	lookupReqMutex    sync.RWMutex
	lookupReqRegistry map[string]map[string]Peer

	deleteWaitMutex         sync.RWMutex
	deleteWaitDataRegistry  map[string]chan string
	deleteWaitErrorRegistry map[string]chan error

	deleteReqMutex    sync.RWMutex
	deleteReqRegistry map[string]map[string]Peer

	conclock sync.RWMutex
}

func NewStdPeer() *stdPeer {
	p := &stdPeer{
		id:                      RandomID(M),
		msgq:                    make(chan Message, QueueSize),
		eventq:                  make(chan Event, QueueSize),
		eventreg:                newmsgcount(),
		init:                    make(chan struct{}),
		data:                    &MapWrapper{data: make(map[string]string)},
		putReqRegistry:          make(map[string]map[string]Peer),
		putWaitDataRegistry:     make(map[string]chan string),
		putWaitErrorRegistry:    make(map[string]chan error),
		lookupReqRegistry:       make(map[string]map[string]Peer),
		lookupWaitDataRegistry:  make(map[string]chan string),
		lookupWaitErrorRegistry: make(map[string]chan error),
		deleteReqRegistry:       make(map[string]map[string]Peer),
		deleteWaitDataRegistry:  make(map[string]chan string),
		deleteWaitErrorRegistry: make(map[string]chan error),
	}
	PeerRegistry.Register(p)
	return p
}

func (p *stdPeer) Kill() {
	p.mutex.Lock()
	p.dead = true
	p.mutex.Unlock()
}

func (p *stdPeer) serve() {

	for i := 0; i < WorkerCount; i++ {
		go func() {
			for {
				select {
				case msg := <-p.msgq:
					if msg == nil {
						fmt.Println("NIL MESSAGE")
					}
					switch msg.Type() {
					case PUT, LOOKUP, DELETE:
						switch msg.Type() {
						case PUT:
							p.addPutReq(msg.(*PutMsg).Key, msg.Sender())
						case LOOKUP:
							p.addLookupReq(msg.(*LookupMsg).Key, msg.Sender())
						case DELETE:
							p.addDeleteReq(msg.(*DeleteMsg).Key, msg.Sender())
						}
						if msg.Sender().ID().eq(msg.Origin().ID()) {
							p.eventq <- msg.getEvent()
							//fmt.Println("SHORT")
						}
						p.eventreg.Register(
							msg.Hash(),
							NewFanInHandlerFactory(
								SafteyValue,
								msg.getEvent(),
								p.eventq,
								Timeout*time.Duration(4+hamming_distance(msg.Sender().ID()[:p.cluster.Dim()], p.GetCluster().Label())),
							),
						)
						p.eventreg.Check(msg)
					case PUTRETURN, LOOKUPRETURN, DELETERETURN:
						p.eventreg.Register(
							msg.Hash(),
							NewFanInHandlerFactory(
								SafteyValue,
								msg.getEvent(),
								p.eventq,
								Timeout*time.Duration(4+hamming_distance(msg.Sender().ID()[:p.cluster.Dim()], p.GetCluster().Label()))*2,
							),
						)
						p.eventreg.Check(msg)
					case JOIN:
						if msg.Sender().GetType() == UNDEF {
							p.eventq <- msg.getEvent()
							continue
						}
						//fmt.Println(msg.Hash(), p.n, p.cluster.Label)
						p.eventreg.Register(
							msg.Hash(),
							NewFanInHandlerFactory(
								SafteyValue,
								msg.getEvent(),
								p.eventq,
								Timeout*time.Duration(4+hamming_distance(msg.Sender().ID()[:p.cluster.Dim()], p.GetCluster().Label())),
							),
						)
						p.eventreg.Check(msg)
					case JOINSPARE, JOINTEMP:
						p.eventreg.Register(
							msg.Hash(),
							NewFanInHandlerFactory(
								SafteyValue,
								msg.getEvent(),
								p.eventq,
								Timeout*time.Duration(4+hamming_distance(msg.Sender().ID()[:p.cluster.Dim()], p.GetCluster().Label())),
							),
						)
						p.eventreg.Check(msg)
					case SPLIT, DISCONNECT, CONNECT:
						p.eventreg.Register(
							msg.Hash(),
							NewFanInHandlerFactory(
								SafteyValue,
								msg.getEvent(),
								p.eventq,
								Timeout*time.Duration(4+hamming_distance(msg.Sender().ID()[:p.cluster.Dim()], p.GetCluster().Label())),
							),
						)
						p.eventreg.Check(msg)
					}
				case event := <-p.eventq:
					done := make(chan struct{})
					fire := func() {
						event.fire(p)
						done <- struct{}{}
					}

					if event.Type() != JOINSPARE {
						p.split.RLock()
					}
					if event.Safe() {
						p.conclock.RLock()
						go fire()
						select {
						case <-done:
						case <-time.After(event.Timeout(p)):
						}
						p.conclock.RUnlock()
					} else {
						//fmt.Println(event.Type())
						p.conclock.Lock()
						go fire()
						select {
						case <-done:
						case <-time.After(event.Timeout(p)):
						}

						p.conclock.Unlock()
					}
					if event.Type() != JOINSPARE {
						p.split.RUnlock()
					}
				}
			}
		}()
	}
}

func (p *stdPeer) lock() {
	p.mutex.Lock()
}

func (p *stdPeer) unlock() {
	p.mutex.Unlock()
}

func (p *stdPeer) GetType() PeerType {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.peerType
}

func (p *stdPeer) SetType(t PeerType) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.peerType = t
}

func (p *stdPeer) ID() ID {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.id
}

func (p *stdPeer) GetCluster() *Cluster {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.cluster
}

func (p *stdPeer) SetCluster(c *Cluster) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.cluster = c
}
func (p *stdPeer) SetClusterType(c *Cluster, t PeerType) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.cluster = c
	p.peerType = t
}

func (p *stdPeer) send(msg Message) {
	go func() {
		time.Sleep(Latency)
		select {
		case p.msgq <- msg:
		case <-time.After(Timeout + Latency):
		}

	}()
}

func (p *stdPeer) ready() {
	select {
	case p.init <- struct{}{}:
	default:
	}
}

func (p *stdPeer) splitlock() {
	p.split.Lock()
}

func (p *stdPeer) unsplitlock() {
	p.split.Unlock()
}

func (p *stdPeer) rsplitlock() {
	//fmt.Println("locked", p.id.String())
	p.split.RLock()
}

func (p *stdPeer) runsplitlock() {
	p.split.RUnlock()
}
func (p *stdPeer) Join(bootstrap *Cluster) *Cluster {
	bootstrap.SendToVc(NewJoin(p, p))
	<-p.init
	p.serve()
	return p.GetCluster()
}

func (p *stdPeer) Leave() {
	p.cluster.replaceVC(p)
	PeerRegistry.Delete(p.ID())
}

func (p *stdPeer) Put(key ID, data string) (string, error) {
	resp, err := p.addPutWait(key)
	msg := NewPut(p, key, data, p)

	//fmt.Println("ATTEMPTED PUT")
	p.GetCluster().SendToVc(msg)
	select {
	case r := <-resp:
		return r, <-err
	case <-time.After(Timeout * time.Duration(4+hamming_distance(key[:p.cluster.Dim()], p.GetCluster().Label())) * 2):
	}
	return "", errors.New("TIMEOUT")
}

func (p *stdPeer) Get(key ID) (string, error) {
	resp, err := p.addLookupWait(key)
	msg := NewLookup(p, key, p)

	//fmt.Println("ATTEMPTED GET")
	p.GetCluster().SendToVc(msg)
	select {
	case r := <-resp:
		return r, <-err
	case <-time.After(Timeout * time.Duration(4+hamming_distance(key[:p.cluster.Dim()], p.GetCluster().Label())) * 2):
	}
	return "", errors.New("TIMEOUT")
}

func (p *stdPeer) Delete(key ID) (string, error) {
	resp, err := p.addDeleteWait(key)
	msg := NewDelete(p, key, p)

	//fmt.Println("ATTEMPTED DELETE")
	p.GetCluster().SendToVc(msg)
	select {
	case r := <-resp:
		return r, <-err
	case <-time.After(Timeout * time.Duration(4+hamming_distance(key[:p.cluster.Dim()], p.GetCluster().Label())) * 2):
	}
	return "", errors.New("TIMEOUT")
}

func (p *stdPeer) put(id ID, data string) bool {
	return p.data.Swap(id.String(), data) == data
}

func (p *stdPeer) get(id ID) (string, bool) {
	return p.data.Get(id.String())
}

func (p *stdPeer) delete(id ID) (string, bool) {
	return p.data.Delete(id.String())
}

func (p *stdPeer) dataStore() DataStore {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.data
}

func (p *stdPeer) replaceData(data DataStore) {
	p.mutex.Lock()
	p.data = data
	p.mutex.Unlock()
}

func (p *stdPeer) flushPutReqs(key ID) (clusters map[string]Peer) {
	p.putReqMutex.Lock()
	clusters = p.putReqRegistry[key.String()]
	delete(p.putReqRegistry, key.String())
	p.putReqMutex.Unlock()
	return
}

func (p *stdPeer) addPutReq(key ID, peer Peer) {
	p.putReqMutex.Lock()
	if _, ok := p.putReqRegistry[key.String()]; !ok {
		p.putReqRegistry[key.String()] = make(map[string]Peer)
	}
	p.putReqRegistry[key.String()][peer.ID().String()] = peer
	p.putReqMutex.Unlock()
}

func (p *stdPeer) flushPutWait(key ID, data string, err error) {
	p.putWaitMutex.Lock()
	datachan, ok := p.putWaitDataRegistry[key.String()]
	errchan := p.putWaitErrorRegistry[key.String()]
	delete(p.putWaitDataRegistry, key.String())
	delete(p.putWaitErrorRegistry, key.String())
	p.putWaitMutex.Unlock()
	if !ok {
		return
	}

	for {
		select {
		case datachan <- data:
		default:
			break
		}
		select {
		case errchan <- err:
		default:
		}
	}
}

func (p *stdPeer) addPutWait(key ID) (chan string, chan error) {
	p.putWaitMutex.Lock()
	datachan, ok := p.putWaitDataRegistry[key.String()]
	errchan, ok := p.putWaitErrorRegistry[key.String()]
	if !ok {
		p.putWaitDataRegistry[key.String()] = make(chan string)
		p.putWaitErrorRegistry[key.String()] = make(chan error)
	}
	datachan, _ = p.putWaitDataRegistry[key.String()]
	errchan, _ = p.putWaitErrorRegistry[key.String()]
	p.putWaitMutex.Unlock()
	return datachan, errchan
}

func (p *stdPeer) flushLookupReqs(key ID) (clusters map[string]Peer) {
	p.lookupReqMutex.Lock()
	clusters = p.lookupReqRegistry[key.String()]
	delete(p.lookupReqRegistry, key.String())
	p.lookupReqMutex.Unlock()
	return
}

func (p *stdPeer) addLookupReq(key ID, peer Peer) {
	p.lookupReqMutex.Lock()
	if _, ok := p.lookupReqRegistry[key.String()]; !ok {
		p.lookupReqRegistry[key.String()] = make(map[string]Peer)
	}
	p.lookupReqRegistry[key.String()][peer.ID().String()] = peer
	p.lookupReqMutex.Unlock()
}

func (p *stdPeer) flushLookupWait(key ID, data string, err error) {
	p.lookupWaitMutex.Lock()
	datachan, ok := p.lookupWaitDataRegistry[key.String()]
	errchan := p.lookupWaitErrorRegistry[key.String()]
	delete(p.lookupWaitDataRegistry, key.String())
	delete(p.lookupWaitErrorRegistry, key.String())
	p.lookupWaitMutex.Unlock()
	if !ok {
		return
	}

	for {
		select {
		case datachan <- data:
		default:
			break
		}
		select {
		case errchan <- err:
		default:
		}
	}
}

func (p *stdPeer) addLookupWait(key ID) (chan string, chan error) {
	p.lookupWaitMutex.Lock()
	datachan, ok := p.lookupWaitDataRegistry[key.String()]
	errchan, ok := p.lookupWaitErrorRegistry[key.String()]
	if !ok {
		p.lookupWaitDataRegistry[key.String()] = make(chan string)
		p.lookupWaitErrorRegistry[key.String()] = make(chan error)
	}
	datachan, _ = p.lookupWaitDataRegistry[key.String()]
	errchan, _ = p.lookupWaitErrorRegistry[key.String()]
	p.lookupWaitMutex.Unlock()
	return datachan, errchan
}

func (p *stdPeer) flushDeleteReqs(key ID) (clusters map[string]Peer) {
	p.deleteReqMutex.Lock()
	clusters = p.deleteReqRegistry[key.String()]
	delete(p.deleteReqRegistry, key.String())
	p.deleteReqMutex.Unlock()
	return
}

func (p *stdPeer) addDeleteReq(key ID, peer Peer) {
	p.deleteReqMutex.Lock()
	if _, ok := p.deleteReqRegistry[key.String()]; !ok {
		p.deleteReqRegistry[key.String()] = make(map[string]Peer)
	}
	p.deleteReqRegistry[key.String()][peer.ID().String()] = peer
	p.deleteReqMutex.Unlock()
}

func (p *stdPeer) flushDeleteWait(key ID, data string, err error) {
	p.deleteWaitMutex.Lock()
	datachan, ok := p.deleteWaitDataRegistry[key.String()]
	errchan := p.deleteWaitErrorRegistry[key.String()]
	delete(p.deleteWaitDataRegistry, key.String())
	delete(p.deleteWaitErrorRegistry, key.String())
	p.deleteWaitMutex.Unlock()
	if !ok {
		return
	}

	for {
		select {
		case datachan <- data:
		default:
			break
		}
		select {
		case errchan <- err:
		default:
		}
	}
}

func (p *stdPeer) addDeleteWait(key ID) (chan string, chan error) {
	p.deleteWaitMutex.Lock()
	datachan, ok := p.deleteWaitDataRegistry[key.String()]
	errchan, ok := p.deleteWaitErrorRegistry[key.String()]
	if !ok {
		p.deleteWaitDataRegistry[key.String()] = make(chan string)
		p.deleteWaitErrorRegistry[key.String()] = make(chan error)
	}
	datachan, _ = p.deleteWaitDataRegistry[key.String()]
	errchan, _ = p.deleteWaitErrorRegistry[key.String()]
	p.deleteWaitMutex.Unlock()
	return datachan, errchan
}
