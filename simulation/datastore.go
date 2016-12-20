package simulation

import "sync"

type DataStore interface {
	Get(string) (string, bool)
	Set(string, string)
	Swap(string, string) string
	Delete(string) (string, bool)
	Copy() DataStore
	Merge(DataStore)
	Filter(ID)
	dataReveal() map[string]string
}

type MapWrapper struct {
	mutex sync.RWMutex
	data  map[string]string
}

func (m *MapWrapper) Set(key, val string) {
	m.mutex.Lock()
	m.data[key] = val
	m.mutex.Unlock()
}

func (m *MapWrapper) Get(key string) (val string, exists bool) {
	m.mutex.RLock()
	val, exists = m.data[key]
	m.mutex.RUnlock()
	return
}

func (m *MapWrapper) Delete(key string) (val string, exists bool) {
	m.mutex.Lock()
	val, exists = m.data[key]
	delete(m.data, key)
	m.mutex.Unlock()
	return
}

func (m *MapWrapper) Swap(key, val string) (old string) {
	m.mutex.Lock()
	old = m.data[key]
	m.data[key] = val
	m.mutex.Unlock()
	return
}

func (m *MapWrapper) dataReveal() map[string]string {
	data := make(map[string]string)
	m.mutex.RLock()
	for key, val := range m.data {
		data[key] = val
	}
	m.mutex.RUnlock()
	return m.data
}

func (m *MapWrapper) Copy() DataStore {
	m2 := &MapWrapper{data: make(map[string]string)}
	m.mutex.RLock()
	for key, val := range m.data {
		m2.data[key] = val
	}
	m.mutex.RUnlock()
	return m2
}

func (m *MapWrapper) Filter(label ID) {
	m.mutex.Lock()
	for key, _ := range m.data {
		if !label.prefix(IDFromString(key)) {
			delete(m.data, key)
		}
	}
	m.mutex.Unlock()
}

func (m *MapWrapper) Merge(m2 DataStore) {
	m.mutex.Lock()
	for key, val := range m2.dataReveal() {
		m.data[key] = val
	}
	m.mutex.Unlock()
}
