package node

import (
	"sync"
)

type KVStore interface {
	Get(string) []byte
	Set(string, []byte)
	Del(string)
}

type MapWrapper struct {
	data map[string][]byte
	sync.RWMutex
}

func NewMapWrapper() *MapWrapper {
	return &MapWrapper{
		data: make(map[string][]byte),
	}
}

func (m *MapWrapper) Get(key string) []byte {
	m.RLock()
	r := m.data[key]
	m.RUnlock()
	return r
}

func (m *MapWrapper) Set(key string, val []byte) {
	m.Lock()
	m.data[key] = val
	m.Unlock()
}

func (m *MapWrapper) Del(key string) {
	m.Lock()
	delete(m.data, key)
	m.Unlock()
}
