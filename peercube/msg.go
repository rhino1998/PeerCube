package peercube

import (
	"encoding/gob"
	"fmt"
)

type MsgType byte

const (
	LOOKUP MsgType = iota
	LOOKUPRETURN
	JOIN
	JOINTEMP
	JOINSPARE
	JOINACK
	SPLIT
	PUT
	PUTRETURN
	CONNECT
	DISCONNECT
)

type Message interface {
	getEvent() Event
	eq(o Message) bool
	Sender() *Peer
	Type() MsgType
	Safe() bool
	Hash() string
	ResponseHash() string
}

func init() {
	gob.Register(&LookupMsg{})
	gob.Register(&JoinMsg{})
	gob.Register(&JoinSpareMsg{})
	gob.Register(&JoinTempMsg{})
	gob.Register(&JoinAckMsg{})
	gob.Register(&SplitMsg{})
	gob.Register(&ConnectInformMsg{})
	gob.Register(&DisconnectInformMsg{})
}

type Msg struct {
	Prev *Peer
}

func (m *Msg) Sender() *Peer {
	return m.Prev
}

type LookupMsg struct {
	Msg
	*LookupEvent
}

func NewLookup(Prev *Peer, key ID, origin *Peer) *LookupMsg {
	return &LookupMsg{
		Msg: Msg{
			Prev: Prev,
		},
		LookupEvent: &LookupEvent{
			Prev:   Prev,
			Origin: origin,
			Key:    key,
		},
	}
}

func (m *LookupMsg) Hash() string {
	return fmt.Sprintf("LOOKUP:%v", m.Key.String())
}

func (m *LookupMsg) ResponseHash() string {
	return fmt.Sprintf("LOOKUPRETURN:%v", m.Key.String())
}

func (m *LookupMsg) getEvent() Event {
	return m.LookupEvent
}

func (m *LookupMsg) eq(o Message) bool {
	return m.Hash() == o.Hash()
}

type JoinMsg struct {
	Msg
	*JoinEvent
}

func NewJoin(q *Peer, origin *Peer) *JoinMsg {
	return &JoinMsg{
		Msg: Msg{
			Prev: q,
		},
		JoinEvent: &JoinEvent{
			Origin: origin,
		},
	}
}

func (m *JoinMsg) Hash() string {
	return fmt.Sprintf("JOIN:%v", m.Origin.ID.String())
}

func (m *JoinMsg) ResponseHash() string {
	return fmt.Sprintf("JOINACK:%v", m.Origin.ID.String())
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

func NewJoinSpare(q *Peer, origin *Peer) *JoinSpareMsg {
	return &JoinSpareMsg{
		JoinSpareEvent: &JoinSpareEvent{
			Origin: origin,
		},
		Msg: Msg{
			Prev: q,
		},
	}
}

func (m *JoinSpareMsg) Hash() string {
	return fmt.Sprintf("JOINSPARE:%v", m.Origin.ID.String())
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

func NewJoinTemp(q *Peer, origin *Peer) *JoinTempMsg {
	return &JoinTempMsg{
		JoinTempEvent: &JoinTempEvent{
			Origin: origin,
		},

		Msg: Msg{
			Prev: q,
		},
	}
}

func (m *JoinTempMsg) Hash() string {
	return fmt.Sprintf("JOINTEMP:%v", m.Origin.ID.String())
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

type JoinAckMsg struct {
	Msg
	*JoinAckEvent
}

func NewJoinAck(q *Peer, state *State) *JoinAckMsg {
	return &JoinAckMsg{
		JoinAckEvent: &JoinAckEvent{
			State: state,
		},
		Msg: Msg{
			Prev: q,
		},
	}
}

func (m *JoinAckMsg) Hash() string {
	return fmt.Sprintf("JOINACK:%v|%v|%v|%v", m.State.Type, m.State.Cluster.Label.String(), len(m.State.Pt), len(m.State.Rt))
}

func (m *JoinAckMsg) ResponseHash() string {
	return ""
}

func (m *JoinAckMsg) eq(o Message) bool {
	return m.Hash() == o.Hash()
}

func (m *JoinAckMsg) getEvent() Event {
	return m.JoinAckEvent
}

type SplitMsg struct {
	Msg
	*SplitEvent
}

//NewSplit initializes a new SPLIT message
func NewSplit(q *Peer, state *State) *SplitMsg {
	return &SplitMsg{
		SplitEvent: &SplitEvent{
			State: state,
		},

		Msg: Msg{
			Prev: q,
		},
	}
}

func (m *SplitMsg) Hash() string {
	return fmt.Sprintf("SPLIT:%v|%v|%v|%v", m.State.Type, m.State.Cluster.Label.String(), len(m.State.Pt), len(m.State.Rt))
}

func (m *SplitMsg) ResponseHash() string {
	return ""
}

func (m *SplitMsg) eq(o Message) bool {
	return m.Hash() == o.Hash()
}

func (m *SplitMsg) getEvent() Event {
	return m.SplitEvent
}

type ConnectInformMsg struct {
	Msg
	*ConnectInformEvent
}

func NewConnectInform(q *Peer, c *Cluster) *ConnectInformMsg {
	return &ConnectInformMsg{
		ConnectInformEvent: &ConnectInformEvent{
			Cluster: c,
		},

		Msg: Msg{
			Prev: q,
		},
	}
}
func (m *ConnectInformMsg) Hash() string {
	return fmt.Sprintf("CONNECTINFORM:%v", m.Cluster.Label.String())
}

func (m *ConnectInformMsg) ResponseHash() string {
	return ""
}

func (m *ConnectInformMsg) eq(o Message) bool {
	return m.Hash() == o.Hash()
}

func (m *ConnectInformMsg) getEvent() Event {
	return m.ConnectInformEvent
}

type DisconnectInformMsg struct {
	Msg
	*DisconnectInformEvent
}

func NewDisconnectInform(q *Peer, c *Cluster) *DisconnectInformMsg {
	return &DisconnectInformMsg{
		DisconnectInformEvent: &DisconnectInformEvent{
			Cluster: c,
		},

		Msg: Msg{
			Prev: q,
		},
	}
}
func (m *DisconnectInformMsg) Hash() string {
	return fmt.Sprintf("DISCONNECTINFORM:%v", m.Cluster.Label.String())
}

func (m *DisconnectInformMsg) ResponseHash() string {
	return ""
}

func (m *DisconnectInformMsg) eq(o Message) bool {
	return m.Hash() == o.Hash()
}

func (m *DisconnectInformMsg) getEvent() Event {
	return m.DisconnectInformEvent
}
