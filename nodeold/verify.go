package node

import "sync"

type RespQuorum struct {
	freq     []uint64
	resps    []*Resp
	resplock *sync.RWMutex
	checks   uint64
	done     chan *Resp
}

func NewRespQuorum(n uint64) *RespQuorum {
	return &RespQuorum{
		checks:   n,
		done:     make(chan *Resp),
		freq:     make([]uint64, 0),
		resps:    make([]*Resp, 0),
		resplock: &sync.RWMutex{},
	}
}

func (q *RespQuorum) Check(resp *Resp) bool {
	q.resplock.Lock()
	defer q.resplock.Unlock()
	for i, r := range q.resps {
		if r.equals(resp) {
			q.freq[i]++
			if q.freq[i] < q.checks {
				return false
			}
			for {
				select {
				case q.done <- resp:
				default:
					return true
				}
			}
		}
	}
	q.resps = append(q.resps, resp)
	q.freq = append(q.freq, 1)
	return false
}

func (q *RespQuorum) Wait() *Resp {
	return <-q.done
}

type RespQuorumRegistry struct {
	quora map[string]*RespQuorum
	sync.RWMutex
}

func NewRespQuorumRegistry() *RespQuorumRegistry {
	return &RespQuorumRegistry{quora: make(map[string]*RespQuorum)}
}

func (r *RespQuorumRegistry) Exists(id ID) bool {
	key := id.String()
	r.RLock()
	_, found := r.quora[key]
	r.RUnlock()
	return found
}

//Register adds a new RespQuorum to the registry or returns false if one already exists
func (r *RespQuorumRegistry) Register(key Key, count uint64) bool {
	r.Lock()
	defer r.Unlock()

	_, found := r.quora[key.Key]
	if found {
		return false
	}
	r.quora[key.Key] = NewRespQuorum(count)
	return true

}

func (r *RespQuorumRegistry) Wait(key Key) *Resp {
	r.RLock()
	q := r.quora[key.Key]
	r.Unlock()
	return q.Wait()
}

func (r *RespQuorumRegistry) Check(key Key, resp *Resp) bool {
	r.Lock()
	q, found := r.quora[key.Key]
	if found {
		return false
	}
	if q.Check(resp) {
		delete(r.quora, key.Key)
	}
	r.Unlock()
	return true
}
