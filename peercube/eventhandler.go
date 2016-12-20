package peercube

import (
	"fmt"
	"sync"
)

type MessageEventHandler interface {
	Wait()
	Trigger(m Message) error
	fire()
	setCleanUp(func())
	cleanUp()
	kill()
}

type InvalidMessageTypeError struct {
	t MsgType
}

func NewInvalidMessageTypeError(t MsgType) InvalidMessageTypeError {
	return InvalidMessageTypeError{t: t}
}

func (e InvalidMessageTypeError) Error() string {
	return fmt.Sprintf("Invalid Message Type for context: %v", e.t)
}

type eventhandler struct {
	mutex  sync.Mutex
	clean  func()
	wg     *sync.WaitGroup
	sema   chan struct{}
	eventq chan<- Event
	quit   bool
	wg2    *sync.WaitGroup
}

func (e *eventhandler) kill() {
	e.quit = true
	for {
		select {
		case e.sema <- struct{}{}:
			e.wg.Done()
		default:
			return
			//Potentially do something else
		}
	}
}

func (e *eventhandler) setCleanUp(clean func()) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.clean = clean
}

func (e *eventhandler) cleanUp() {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	if e.clean != nil {
		e.clean()
	}
}

func (e *eventhandler) Wait() {
	e.wg2.Wait()
}

type FanInHandler struct {
	eventhandler
	event Event
}

func NewFanInHandler(n int, event Event, eventq chan<- Event) *FanInHandler {
	h := &FanInHandler{

		event: event,
		eventhandler: eventhandler{
			wg:     &sync.WaitGroup{},
			wg2:    &sync.WaitGroup{},
			sema:   make(chan struct{}, n),
			eventq: eventq,
		},
	}
	h.wg2.Add(1)
	h.wg.Add(n)
	go func() {
		h.wg.Wait()
		h.fire()
		h.wg2.Done()
	}()
	return h
}

func (h *FanInHandler) Trigger(m Message) error {
	select {
	case h.sema <- struct{}{}:
		h.wg.Done()
	default:
		return nil
		//Potentially do something else
	}
	return nil
}

func (h *FanInHandler) fire() {
	if !h.quit {
		h.eventq <- h.event
	}
	h.cleanUp()
	return
}

type FanInOutHandler struct {
	eventhandler
	events map[string]Event
	mutex  *sync.RWMutex
}

func NewFanInOutHandler(n int, eventq chan<- Event) *FanInOutHandler {
	h := &FanInOutHandler{
		eventhandler: eventhandler{
			wg:     &sync.WaitGroup{},
			sema:   make(chan struct{}, n),
			eventq: eventq,
		},
		events: make(map[string]Event),
		mutex:  &sync.RWMutex{},
	}
	h.wg.Add(n)
	go func() {
		h.wg.Wait()
		h.fire()
	}()
	return h
}

func (h *FanInOutHandler) Trigger(m Message) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.events[m.Sender().ID.String()] = m.getEvent()
	select {
	case h.sema <- struct{}{}:
		h.wg.Done()
	default:
		//Potentially do something else
	}
	return nil
}

func (h *FanInOutHandler) fire() {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	for _, event := range h.events {
		h.eventq <- event
	}
	h.cleanUp()
}

type JoinAckHandler struct {
	eventhandler
	n     int
	acks  map[string]int
	event Event
	msgs  map[string]Message
	mutex *sync.RWMutex
}

func NewJoinAckHandler(n int, eventq chan<- Event) *JoinAckHandler {
	h := &JoinAckHandler{
		eventhandler: eventhandler{
			wg:     &sync.WaitGroup{},
			wg2:    &sync.WaitGroup{},
			sema:   make(chan struct{}, 1),
			eventq: eventq,
		},
		n:     n,
		acks:  make(map[string]int),
		msgs:  make(map[string]Message),
		mutex: &sync.RWMutex{},
	}
	h.wg2.Add(1)
	h.wg.Add(1)
	go func() {
		h.wg.Wait()
		h.fire()
		h.wg2.Done()
	}()
	return h
}

func (h *JoinAckHandler) Trigger(m Message) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	if m.Type() != JOINACK {
		return NewInvalidMessageTypeError(m.Type())
	}
	h.acks[m.Hash()]++
	if h.acks[m.Hash()] < h.n {
		return nil
	}

	select {
	case h.sema <- struct{}{}:
		h.event = m.getEvent()
		h.wg.Done()
	default:
		//Potentially do something else
	}
	return nil
}

func (h *JoinAckHandler) fire() {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.eventq <- h.event
	h.cleanUp()
}
