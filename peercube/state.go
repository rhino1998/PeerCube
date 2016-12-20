package peercube

import "sync"

type State struct {
	Cluster *Cluster
	Data    map[string][]byte
	Type    PeerType
	Rt      []*Cluster
	Pt      map[string]*Cluster
}

func (s *State) equals(o *State) bool {
	return s.Type == o.Type && s.Cluster.Label.eq(o.Cluster.Label)
}

type StateQuorum struct {
	freq     []uint64
	resps    []*State
	resplock *sync.RWMutex
	checks   uint64
	done     chan *State
}

func NewStateQuorum(n uint64) *StateQuorum {
	return &StateQuorum{
		checks:   n,
		done:     make(chan *State),
		freq:     make([]uint64, 0),
		resps:    make([]*State, 0),
		resplock: &sync.RWMutex{},
	}
}

func (q *StateQuorum) Check(state *State) bool {
	q.resplock.Lock()
	defer q.resplock.Unlock()
	for i, r := range q.resps {
		if r.equals(state) {
			q.freq[i]++
			if q.freq[i] < q.checks {
				return false
			}
			for {
				select {
				case q.done <- state:
				default:
					return true
				}
			}
		}
	}
	q.resps = append(q.resps, state)
	q.freq = append(q.freq, 1)
	return false
}

//Wait returns the agreed value based on the quorum
func (q *StateQuorum) Wait() *State {
	return <-q.done
}

//Done returns the channel that signals a complete quorum
func (q *StateQuorum) Done() <-chan *State {
	return q.done
}
