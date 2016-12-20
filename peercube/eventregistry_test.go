package peercube

import "testing"

func TestEventRegistry(t *testing.T) {
	eventq := make(chan Event)
	Msg := &TestMsg{
		TestEvent: &TestEvent{
			val: 2,
		},
		Msg: Msg{
			Prev: &Peer{
				ID: RandomID(M),
			},
		},
	}
	fanin := NewFanInHandler(5, nil, eventq)
	er := newmsgcount()
	er.Register(Msg.Hash(), fanin)
	er.Check(Msg)
	er.Check(Msg)
	er.Check(Msg)
	er.Check(Msg)
	er.Check(Msg)
	er.Check(Msg)
	<-eventq
}
