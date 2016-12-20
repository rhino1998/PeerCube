package simulation

import (
	"fmt"
	"sync"
	"time"
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
	mutex     sync.RWMutex
	cleanOnce sync.Once
	clean     func()
	actOnce   sync.Once
	sema      chan struct{}
	eventq    chan<- Event
	quit      bool
	done      bool
	wg        sync.WaitGroup
}

func (e *eventhandler) Wait() {
	e.wg.Wait()
}

func (e *eventhandler) kill() {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.quit = true
	for {
		select {
		case e.sema <- struct{}{}:
		default:
			e.cleanUp()
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
	if e.clean != nil {
		e.cleanOnce.Do(e.clean)
	}
}

type FanInHandler struct {
	eventhandler
	event Event
}

func NewFanInHandlerFactory(n int, event Event, eventq chan<- Event, timeout time.Duration) func() MessageEventHandler {
	return func() MessageEventHandler {

		h := &FanInHandler{
			event: event,
			eventhandler: eventhandler{
				sema:   make(chan struct{}, n),
				eventq: eventq,
			},
		}
		h.wg.Add(1)
		go func() {
			select {
			case <-time.After(timeout):
				//fmt.Println("time", timeout)
				h.mutex.Lock()
				defer h.mutex.Unlock()
				if h.done {
					return
				}
				h.quit = true
				go h.cleanUp()

				h.wg.Done()

			}
		}()
		return h
	}
}

func (h *FanInHandler) Trigger(m Message) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	if h.quit || h.done {
		return nil
	}
	select {
	case h.sema <- struct{}{}:
		return nil
	default:
		h.done = true
		go h.fire()
		go h.cleanUp()
		return nil
		//Potentially do something else
	}
	return nil
}

func (h *FanInHandler) act() {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	if !h.quit {
		h.eventq <- h.event
	}
	h.wg.Done()
}

func (h *FanInHandler) fire() {
	h.actOnce.Do(h.act)
	return
}

type FanInOutHandler struct {
	eventhandler
	events map[string]Event
}

func NewFanInOutHandlerFactory(n int, eventq chan<- Event, timeout time.Duration) func() MessageEventHandler {
	return func() MessageEventHandler {
		h := &FanInOutHandler{
			eventhandler: eventhandler{
				sema:   make(chan struct{}, n),
				eventq: eventq,
			},
			events: make(map[string]Event),
		}
		h.wg.Add(1)
		go func() {
			select {
			case <-time.After(timeout):
				//fmt.Println("time", timeout)
				h.mutex.Lock()
				defer h.mutex.Unlock()
				if h.done {
					return
				}
				h.quit = true
				go h.cleanUp()

				h.wg.Done()

			}
		}()
		return h
	}
}

func (h *FanInOutHandler) Trigger(m Message) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	if h.quit || h.done {
		return nil
	}
	h.events[m.Sender().ID().String()] = m.getEvent()
	h.done = true
	select {
	case h.sema <- struct{}{}:
	default:
		h.fire()
		h.wg.Done()
		return nil
		//Potentially do something else
	}
	return nil
}

func (h *FanInOutHandler) fire() {
	for _, event := range h.events {
		h.eventq <- event
	}
	h.cleanUp()
}
