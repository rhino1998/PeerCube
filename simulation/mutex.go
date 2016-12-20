package simulation

import "sync"

type CheckableMutex struct {
	ext   sync.Mutex
	mutex sync.Mutex
	once  sync.Once
}

func (m CheckableMutex) Lock() {
	m.ext.Lock()
	m.mutex.Lock()
	m.once = sync.Once{}
	m.ext.Unlock()
}

func (m CheckableMutex) Unlock() {
	m.ext.Lock()
	m.once.Do(func() {
		m.mutex.Unlock()
	})
	m.ext.Unlock()
}

type CheckableRWMutex struct {
	ext   sync.Mutex
	mutex sync.RWMutex
	once  sync.Once
}

func (m *CheckableRWMutex) Lock() {
	m.ext.Lock()
	m.mutex.Lock()
	m.once = sync.Once{}
	m.ext.Unlock()
}

func (m *CheckableRWMutex) Unlock() {
	m.ext.Lock()
	m.once.Do(m.mutex.Unlock)
	m.ext.Unlock()
}

func (m *CheckableRWMutex) RUnlock() {
	m.ext.Lock()
	m.mutex.RUnlock()
	m.ext.Unlock()
}

func (m *CheckableRWMutex) RLock() {
	m.ext.Lock()
	m.mutex.RLock()
	m.ext.Unlock()
}
