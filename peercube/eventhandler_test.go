package peercube

import (
	"fmt"
	"sync"
	"testing"
)

type TestEvent struct {
	val int
}

func (e *TestEvent) fire(p *Peer) error {
	fmt.Println(e.val)
	return nil
}
func (e *TestEvent) Safe() bool {
	return true
}
func (e *TestEvent) Type() MsgType {
	return 0
}

type TestMsg struct {
	Msg
	*TestEvent
}

func (m *TestMsg) Hash() string {
	return fmt.Sprintf("%v", m.val)
}

func (m *TestMsg) ResponseHash() string {
	return fmt.Sprintf("%v", m.val)
}

func (m *TestMsg) eq(o Message) bool {
	return m.Hash() == o.Hash()
}
func (m *TestMsg) getEvent() Event {
	return m.TestEvent
}

func TestFanIn(t *testing.T) {
	eventq := make(chan Event)
	fanin := NewFanInHandler(5, &TestEvent{}, eventq)
	fanin.Trigger(nil)
	fanin.Trigger(nil)
	fanin.Trigger(nil)
	fanin.Trigger(nil)
	fanin.Trigger(nil)
	fanin.Trigger(nil)
	fanin.Trigger(nil)
	fanin.Trigger(nil)
	(<-eventq).fire(nil)
}

func TestFanInOut(t *testing.T) {
	n := 7
	wg := &sync.WaitGroup{}
	eventq := make(chan Event)
	fanin := NewFanInOutHandler(n, eventq)
	for i := 0; i < n; i++ {
		fanin.Trigger(&TestMsg{
			TestEvent: &TestEvent{
				val: i,
			},
			Msg: Msg{
				Prev: &Peer{
					ID: RandomID(M),
				},
			},
		})
		wg.Add(1)
		go func() {
			(<-eventq).fire(nil)
			wg.Done()
		}()
	}
	wg.Wait()

}
