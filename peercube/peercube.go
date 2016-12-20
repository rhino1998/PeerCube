package peercube

import (
	"sync"
	"time"
)

const (
	M           uint64        = 24
	SMax        uint64        = 12
	SMin        uint64        = 4
	TSplit      uint64        = 6
	Timeout     time.Duration = 500 * time.Millisecond
	QueueSize   uint64        = 32
	WorkerCount uint64        = 4
)

//Fako enum thingy
const (
	UNDEF PeerType = iota
	CORE
	TEMP
	SPARE
)

var (
	peerregistry *PeerRegistry
)

func init() {
	//Simulation Things
	peerregistry = NewPeerRegistry()
}

type PeerRegistry struct {
	data map[string]*Peer
	sync.RWMutex
}

func NewPeerRegistry() *PeerRegistry {
	return &PeerRegistry{
		data: make(map[string]*Peer),
	}
}

func (m *PeerRegistry) Get(key string) *Peer {
	m.RLock()
	r := m.data[key]
	m.RUnlock()
	return r
}

func (m *PeerRegistry) Set(key string, peer *Peer) {
	m.Lock()
	m.data[key] = peer
	m.Unlock()
}

func (m *PeerRegistry) Del(key string) {
	m.Lock()
	delete(m.data, key)
	m.Unlock()
}

func (m *PeerRegistry) Length() int {
	m.RLock()
	l := len(m.data)
	m.RUnlock()
	return l
}
