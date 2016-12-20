package simulation

import "sync"

type EventRegistry interface {
	Register(hash string, f func() MessageEventHandler) bool
	Check(m Message) bool
	Wait(hash string)
}

type msgcount struct {
	handlers map[string]MessageEventHandler
	mutex    *sync.RWMutex
}

func newmsgcount() *msgcount {
	return &msgcount{
		handlers: make(map[string]MessageEventHandler),
		mutex:    &sync.RWMutex{},
	}
}

func (m *msgcount) Register(hash string, f func() MessageEventHandler) bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if _, found := m.handlers[hash]; found {
		return false
	}
	h := f()
	m.handlers[hash] = h
	h.setCleanUp(func() {
		m.mutex.Lock()
		defer m.mutex.Unlock()
		delete(m.handlers, hash)
	})
	// /fmt.Println("successful register")
	return true
}

func (m *msgcount) Wait(hash string) {
	m.mutex.RLock()
	wg := m.handlers[hash]
	m.mutex.RUnlock()
	if wg == nil {
		return
	}
	wg.Wait()
}

func (m *msgcount) Check(msg Message) bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	hash := msg.Hash()
	_, found := m.handlers[hash]
	if !found {
		return false
	}
	m.handlers[hash].Trigger(msg)
	return true
}
