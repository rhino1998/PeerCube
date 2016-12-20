package simulation

import "fmt"

type MsgType byte

const (
	LOOKUP MsgType = iota
	LOOKUPRETURN
	PUT
	PUTRETURN
	DELETE
	DELETERETURN
	JOIN
	JOINTEMP
	JOINSPARE
	JOINACK
	SPLIT
	CONNECT
	DISCONNECT
)

type Message interface {
	getEvent() Event
	eq(o Message) bool
	Sender() Peer
	Origin() Peer
	Type() MsgType
	Safe() bool
	Hash() string
}

type Msg struct {
}

type LookupMsg struct {
	Msg
	*LookupEvent
}

func NewLookup(q Peer, key ID, origin Peer) *LookupMsg {
	return &LookupMsg{
		Msg: Msg{},
		LookupEvent: &LookupEvent{
			event: event{
				origin: origin,
				prev:   q,
			},
			Key: key,
		},
	}
}

func (m *LookupMsg) Hash() string {
	return fmt.Sprintf("LOOKUP:%v", m.Key.String())
}

func (m *LookupMsg) getEvent() Event {
	return m.LookupEvent
}

func (m *LookupMsg) eq(o Message) bool {
	return m.Hash() == o.Hash()
}

type LookupReturnMsg struct {
	Msg
	*LookupReturnEvent
}

func NewLookupReturn(q Peer, target ID, key ID, data string, err error, origin Peer) *LookupReturnMsg {
	return &LookupReturnMsg{
		Msg: Msg{},
		LookupReturnEvent: &LookupReturnEvent{
			event: event{
				origin: origin,
				prev:   q,
			},
			Target: target,
			Key:    key,
			Data:   data,
			Error:  err,
		},
	}
}

func (m *LookupReturnMsg) Hash() string {
	return fmt.Sprintf("LOOKUPRETURN:%v", m.Key.String())
}

func (m *LookupReturnMsg) getEvent() Event {
	return m.LookupReturnEvent
}

func (m *LookupReturnMsg) eq(o Message) bool {
	return m.Hash() == o.Hash()
}

type PutMsg struct {
	Msg
	*PutEvent
}

func NewPut(q Peer, key ID, data string, origin Peer) *PutMsg {
	return &PutMsg{
		Msg: Msg{},
		PutEvent: &PutEvent{
			event: event{
				origin: origin,
				prev:   q,
			},
			Key:  key,
			Data: data,
		},
	}
}

func (m *PutMsg) Hash() string {
	return fmt.Sprintf("PUT:%v", m.Key.String())
}

func (m *PutMsg) getEvent() Event {
	return m.PutEvent
}

func (m *PutMsg) eq(o Message) bool {
	return m.Hash() == o.Hash()
}

type PutReturnMsg struct {
	Msg
	*PutReturnEvent
}

func NewPutReturn(q Peer, target ID, key ID, data string, err error, origin Peer) *PutReturnMsg {
	return &PutReturnMsg{
		Msg: Msg{},
		PutReturnEvent: &PutReturnEvent{
			event: event{
				origin: origin,
				prev:   q,
			},
			Target: target,
			Key:    key,
			Data:   data,
			Error:  err,
		},
	}
}

func (m *PutReturnMsg) Hash() string {
	return fmt.Sprintf("PUTRETURN:%v", m.Key.String())
}

func (m *PutReturnMsg) getEvent() Event {
	return m.PutReturnEvent
}

func (m *PutReturnMsg) eq(o Message) bool {
	return m.Hash() == o.Hash()
}

type DeleteMsg struct {
	Msg
	*DeleteEvent
}

func NewDelete(q Peer, key ID, origin Peer) *DeleteMsg {
	return &DeleteMsg{
		Msg: Msg{},
		DeleteEvent: &DeleteEvent{
			event: event{
				origin: origin,
				prev:   q,
			},
			Key: key,
		},
	}
}

func (m *DeleteMsg) Hash() string {
	return fmt.Sprintf("DELETE:%v", m.Key.String())
}

func (m *DeleteMsg) getEvent() Event {
	return m.DeleteEvent
}

func (m *DeleteMsg) eq(o Message) bool {
	return m.Hash() == o.Hash()
}

type DeleteReturnMsg struct {
	Msg
	*DeleteReturnEvent
}

func NewDeleteReturn(q Peer, target ID, key ID, data string, err error, origin Peer) *DeleteReturnMsg {
	return &DeleteReturnMsg{
		Msg: Msg{},
		DeleteReturnEvent: &DeleteReturnEvent{
			event: event{
				origin: origin,
				prev:   q,
			},
			Target: target,
			Key:    key,
			Data:   data,
			Error:  err,
		},
	}
}

func (m *DeleteReturnMsg) Hash() string {
	return fmt.Sprintf("DELETERETURN:%v", m.Key.String())
}

func (m *DeleteReturnMsg) getEvent() Event {
	return m.DeleteReturnEvent
}

func (m *DeleteReturnMsg) eq(o Message) bool {
	return m.Hash() == o.Hash()
}

type JoinMsg struct {
	Msg
	*JoinEvent
}

func NewJoin(q Peer, origin Peer) *JoinMsg {
	return &JoinMsg{
		Msg: Msg{},
		JoinEvent: &JoinEvent{
			event{
				origin: origin,
				prev:   q,
			},
		},
	}
}

func (m *JoinMsg) Hash() string {
	return fmt.Sprintf("JOIN:%v", m.origin.ID().String())
}

func (m *JoinMsg) eq(o Message) bool {
	return m.Hash() == o.Hash()
}

func (j *JoinMsg) getEvent() Event {
	return j.JoinEvent
}

type JoinSpareMsg struct {
	Msg
	*JoinSpareEvent
}

func NewJoinSpare(q Peer, origin Peer) *JoinSpareMsg {
	return &JoinSpareMsg{
		JoinSpareEvent: &JoinSpareEvent{
			event{
				origin: origin,
				prev:   q,
			},
		},
		Msg: Msg{},
	}
}

func (m *JoinSpareMsg) Hash() string {
	return fmt.Sprintf("JOINSPARE:%v", m.origin.ID().String())
}

func (m *JoinSpareMsg) ResponseHash() string {
	return ""

}
func (m *JoinSpareMsg) getEvent() Event {
	return m.JoinSpareEvent
}

func (m *JoinSpareMsg) eq(o Message) bool {
	return m.Hash() == o.Hash()
}

type JoinTempMsg struct {
	Msg
	*JoinTempEvent
}

func NewJoinTemp(q Peer, origin Peer) *JoinTempMsg {
	return &JoinTempMsg{
		JoinTempEvent: &JoinTempEvent{
			event{
				origin: origin,
				prev:   q,
			},
		},

		Msg: Msg{},
	}
}

func (m *JoinTempMsg) Hash() string {
	return fmt.Sprintf("JOINTEMP:%v", m.origin.ID().String())
}

func (m *JoinTempMsg) ResponseHash() string {
	return ""
}

func (m *JoinTempMsg) getEvent() Event {
	return m.JoinTempEvent
}

func (m *JoinTempMsg) eq(o Message) bool {
	return false
}
