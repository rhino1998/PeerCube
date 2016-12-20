package simulation

import (
	"errors"
	"time"
)

type Event interface {
	fire(p Peer) error
	Safe() bool
	Type() MsgType
	Timeout(Peer) time.Duration
}

type event struct {
	prev   Peer
	origin Peer
}

func (m *event) Sender() Peer {
	return m.prev
}

func (m *event) Origin() Peer {
	return m.origin
}

func (e *event) Timeout(p Peer) time.Duration {
	return Timeout * time.Duration(hamming_distance(e.origin.ID(), p.GetCluster().Label()))
}

type LookupEvent struct {
	event
	Key ID
}

func (e *LookupEvent) Safe() bool {
	return true
}

func (e *LookupEvent) Type() MsgType {
	return LOOKUP
}

func (e *LookupEvent) fire(p Peer) error {
	c := p.GetCluster().getClosestCluster(e.Key)

	if c.Label().eq(p.GetCluster().Label()) {
		v, exists := p.get(e.Key)
		for _, pt := range p.flushLookupReqs(e.Key) {
			if exists {
				pt.GetCluster().SendToVc(NewLookupReturn(p, pt.ID(), e.Key, v, nil, p))
			} else {
				pt.GetCluster().SendToVc(NewLookupReturn(p, pt.ID(), e.Key, "", errors.New("DoesNotExist"), p))
			}

		}
		return nil
	}
	c.SendToVc(NewLookup(p, e.Key, e.origin))
	return nil
}

type LookupReturnEvent struct {
	event
	Key    ID
	Data   string
	Error  error
	Target ID
}

func (e *LookupReturnEvent) Safe() bool {
	return true
}

func (e *LookupReturnEvent) Type() MsgType {
	return LOOKUPRETURN
}

func (e *LookupReturnEvent) fire(p Peer) error {
	c := p.GetCluster().getClosestCluster(e.Target)
	if e.Target.prefix(p.ID()) {
		p.flushLookupWait(e.Key, e.Data, e.Error)
	}

	if c.Label().eq(e.prev.GetCluster().Label()) {
		return nil
	}
	for _, pt := range p.flushLookupReqs(e.Key) {
		pt.GetCluster().SendToVc(NewLookupReturn(p, pt.ID(), e.Key, e.Data, e.Error, p))
	}
	return nil
}

type PutEvent struct {
	event
	Key  ID
	Data string
}

func (e *PutEvent) Safe() bool {
	return true
}

func (e *PutEvent) Type() MsgType {
	return PUT
}

func (e *PutEvent) fire(p Peer) error {
	c := p.GetCluster().getClosestCluster(e.Key)

	if c.Label().eq(p.GetCluster().Label()) {
		if !p.put(e.Key, e.Data) {
			for _, pt := range p.flushPutReqs(e.Key) {
				pt.GetCluster().SendToVc(NewPutReturn(p, pt.ID(), e.Key, e.Data, nil, p))
			}
		}
		return nil
	}
	c.SendToVc(NewPut(p, e.Key, e.Data, e.origin))
	return nil
}

type PutReturnEvent struct {
	event
	Key    ID
	Data   string
	Error  error
	Target ID
}

func (e *PutReturnEvent) Safe() bool {
	return true
}

func (e *PutReturnEvent) Type() MsgType {
	return PUTRETURN
}

func (e *PutReturnEvent) fire(p Peer) error {
	c := p.GetCluster().getClosestCluster(e.Target)
	if e.Target.prefix(p.ID()) {
		p.flushPutWait(e.Key, e.Data, nil)
	}

	if c.Label().eq(e.prev.GetCluster().Label()) {
		return nil
	} else {
		for _, pt := range p.flushPutReqs(e.Key) {
			pt.GetCluster().SendToVc(NewPutReturn(p, pt.ID(), e.Key, e.Data, nil, p))
		}
	}
	return nil
}

type DeleteEvent struct {
	event
	Key ID
}

func (e *DeleteEvent) Safe() bool {
	return true
}

func (e *DeleteEvent) Type() MsgType {
	return DELETE
}

func (e *DeleteEvent) fire(p Peer) error {
	c := p.GetCluster().getClosestCluster(e.Key)

	if c.Label().eq(p.GetCluster().Label()) {
		v, exists := p.delete(e.Key)
		for _, pt := range p.flushDeleteReqs(e.Key) {
			if exists {
				pt.GetCluster().SendToVc(NewDeleteReturn(p, pt.ID(), e.Key, v, nil, p))
			}

		}
		return nil
	}
	c.SendToVc(NewDelete(p, e.Key, e.origin))

	return nil
}

type DeleteReturnEvent struct {
	event
	Key    ID
	Data   string
	Error  error
	Target ID
}

func (e *DeleteReturnEvent) Safe() bool {
	return true
}

func (e *DeleteReturnEvent) Type() MsgType {
	return DELETERETURN
}

func (e *DeleteReturnEvent) fire(p Peer) error {
	c := p.GetCluster().getClosestCluster(e.Target)
	if e.Target.prefix(p.ID()) {
		p.flushDeleteWait(e.Key, e.Data, e.Error)
	}

	if c.Label().eq(e.prev.GetCluster().Label()) {
		return nil
	}
	for _, pt := range p.flushDeleteReqs(e.Key) {
		pt.GetCluster().SendToVc(NewDeleteReturn(p, pt.ID(), e.Key, e.Data, e.Error, p))
	}
	return nil
}

type JoinEvent struct {
	event
}

func (e *JoinEvent) Safe() bool {
	return false
}

func (e *JoinEvent) Type() MsgType {
	return JOIN
}
func (e *JoinEvent) fire(p Peer) error {

	c := p.GetCluster().getClosestCluster(e.origin.ID())
	//fmt.Println(p.GetCluster().Label().String(), "nextHop", c.Label().String())
	if p.GetCluster().Label().eq(c.Label()) {
		if c.Label().prefix(e.origin.ID()) {
			c.SendToVc(NewJoinSpare(p, e.origin))
			return nil
		}
		c.SendToVc(NewJoinTemp(p, e.origin))
		return nil
	}
	c.SendToVc(NewJoin(p, e.origin))
	return nil
}

type JoinSpareEvent struct {
	event
}

func (e *JoinSpareEvent) Safe() bool {
	return false
}

func (e *JoinSpareEvent) Type() MsgType {
	return JOINSPARE
}

func (e *JoinSpareEvent) fire(p Peer) error {
	p.GetCluster().Add(e.origin, SPARE)
	return nil
}

type JoinTempEvent struct {
	event
}

func (e *JoinTempEvent) Safe() bool {
	return false
}

func (e *JoinTempEvent) Type() MsgType {
	return JOINTEMP
}

func (e *JoinTempEvent) fire(p Peer) error {
	return nil
}
