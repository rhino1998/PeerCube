package node

import (
	"log"
	"net"
	"net/rpc"
	"sync"
	"time"
)

//Fako enum thingy
const (
	CORE  = byte(iota)
	TEMP  = byte(iota)
	SPARE = byte(iota)
)

type Peer struct {
	//General info
	ID      ID
	TCPAddr *net.TCPAddr
	Type    byte

	//Useful
	timeout time.Duration

	//For Serving
	cluster   *Cluster
	rt        []*Cluster
	data      KVStore
	splitlock *sync.RWMutex

	respquora *RespQuorumRegistry

	//For Clienting
	conn *rpc.Client
}

func NewServerNode(laddr string, M, SMin, SMax, TSplit uint64) (*Peer, error) {
	addr, err := net.ResolveTCPAddr("tcp", laddr)
	if err != nil {
		return nil, err
	}
	p := &Peer{
		ID:      RandomID(M),
		TCPAddr: addr,
	}
	p.cluster = &Cluster{
		M:      M,
		Dim:    0,
		Label:  make(ID, 0),
		SMin:   SMin,
		SMax:   SMax,
		TSplit: TSplit,
		CoreSet: map[string]*Peer{
			p.ID.String(): p,
		},
		SpareSet: make(map[string]*Peer),
		TempSet:  make(map[string]*Peer),
	}
	return p, nil
}

//startRPC starts an rpc server at the local node
//Local Method
func (p *Peer) serve() {
	server := rpc.NewServer()
	server.RegisterName("Peer", p)
	l, e := net.ListenTCP("tcp", p.TCPAddr)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				log.Fatal(err)
			}

			go server.ServeConn(conn)
		}
	}()
}

func (p *Peer) connect() error {
	conn, err := net.DialTCP("tcp", p.TCPAddr, nil)
	if err != nil {
		return err
	}
	p.conn = rpc.NewClient(conn)
	return nil
}

func (p *Peer) ensureConnected() error {
	if p.conn != nil {
		err := p.connect()
		return err
	}
	return nil
}

func (p *Peer) findClosestCluster(id ID) *Cluster {
	if p.cluster.Dim == 0 || p.cluster.Label.prefix(id) {
		return p.cluster
	} else {
		c := p.rt[0]
		for i := uint64(0); i < p.cluster.Dim; i++ {
			if distance(id, p.rt[i].Label).lt(distance(id, c.Label)) {
				c = p.rt[i]
			}
		}
		return c
	}
	return nil
}

func (p *Peer) get(key ID) (resp *Resp, err error) {
	err = p.ensureConnected()
	if err != nil {
		return
	}
	err = p.conn.Call("Peer.Get", &key, resp)
	return
}

func (p *Peer) getCluster() (cluster *Cluster, err error) {
	err = p.conn.Call("Peer.GetCluster", &struct{}{}, cluster)
	return
}

func (p *Peer) join() (*State, error) {
	var state *State
	err := p.conn.Call("Peer.Join", p, state)
	return state, err
}

func (p *Peer) joinByAddr(raddr string) error {
	conn, err := net.Dial("tcp", raddr)
	if err != nil {
		return err
	}
	remote := rpc.NewClient(conn)
	var cluster Cluster
	err = remote.Call("Peer.GetCluster", &struct{}{}, &cluster)
	if err != nil {
		return err
	}
	remote.Close()

	for _, peer := range cluster.CoreSet {
		peer.join()
		log.Println("Joining", peer.ID.String())
	}
	return nil
}

func (p *Peer) split() {

}

//RPC METHODS

func (p *Peer) Join(q *Peer, state *State) error {
	c := p.findClosestCluster(q.ID)

	//Dynamic timeout multiplier
	timeMult := time.Duration(hamming_distance(c.Label, (q.ID)[:len(c.Label)])) + 2

	//Create a quorum listener
	sq := NewStateQuorum((p.cluster.SMin - 1/3 + 1))
	if p.cluster.Label.eq(c.Label) {
		if p.cluster.Label.prefix(q.ID) {
			for _, peer := range p.cluster.CoreSet {
				go func(peer *Peer) {
					var state *State
					call := peer.conn.Go("Peer.JoinSpare", q, state, nil)

					select {
					case <-call.Done:
						sq.Check(state)
						return
					case <-sq.Done():
						return
					case <-time.After(p.timeout * timeMult):
						return
					}
				}(peer)
			}
		} else {
			for _, peer := range p.cluster.CoreSet {
				go func(peer *Peer) {
					var state *State
					call := peer.conn.Go("Peer.JoinTemp", q, state, nil)

					select {
					case <-call.Done:
						sq.Check(state)
						return
					case <-sq.Done():
						return
					case <-time.After(p.timeout * timeMult):
						return
					}
				}(peer)
			}
		}
		*state = *sq.Wait()
		return nil
	}
	for _, peer := range p.cluster.CoreSet {
		go func(peer *Peer) {
			var state *State
			call := peer.conn.Go("Peer.Join", q, state, nil)

			select {
			case <-call.Done:
				sq.Check(state)
				return
			case <-sq.Done():
				return
			case <-time.After(p.timeout * timeMult):
				return
			}
		}(peer)
	}
	*state = *sq.Wait()
	return nil
}
func (p *Peer) JoinSpare(q *Peer, state *State) error {
	p.cluster.mutex.Lock()
	p.cluster.SpareSet[q.ID.String()] = q
	if p.cluster.isSplit() {
		p.split()
	}

	p.cluster.mutex.Unlock()
	return nil
}

func (p *Peer) JoinTemp(q *Peer, state *State) error {
	p.cluster.mutex.Lock()
	p.cluster.SpareSet[q.ID.String()] = q
	if p.cluster.tempIsSplit() {
	}
	*state = State{
		Cluster: p.findClosestCluster(q.ID),
		Type:    TEMP,
	}
	p.cluster.mutex.Unlock()
	return nil
}

func (p *Peer) GetCluster(_ *struct{}, c *Cluster) error {
	*c = *p.cluster
	return nil
}

func (p *Peer) Get(key *ID, resp *Resp) error {
	c := p.findClosestCluster(*key)
	timeMult := time.Duration(hamming_distance(c.Label, (*key)[:len(c.Label)])) + 2

	if p.cluster.Label.eq(c.Label) {

		//Ensure following operation not done twice
		if p.respquora.Register(key, (p.cluster.SMin-1)/3+1) {
			//If not done before, send to all of coreset
			//Broadcast, not implemented exactly yet, fudging it for now

			for _, peer := range p.cluster.CoreSet {

				go func(peer *Peer) {
					var resp *Resp
					call := peer.conn.Go("Peer.Get", key, *resp, nil)

					select {
					case <-call.Done:
						p.respquora.Check(key, resp)
						return
					case <-p.respquora.Done(key):
						return
					case <-time.After(p.timeout * timeMult):
						return
					}
				}(peer)
			}
			*resp = *p.respquora.Wait(key)
			return nil
		}

		*resp = Resp{
			Key:   *key,
			Label: p.cluster.Label,
			Value: p.data.Get(key.String()),
		}
		return nil
	}
	if !p.respquora.Register(key, (p.cluster.SMin-1)/3+1) {
		for _, peer := range p.cluster.CoreSet {
			go func(peer *Peer, key *ID) {
				var resp *Resp
				call := peer.conn.Go("Peer.Get", key, *resp, nil)

				select {
				case <-call.Done:
					p.respquora.Check(key, resp)
					return
				case <-p.respquora.Done(key):
					return
				case <-time.After(p.timeout * timeMult):
					return
				}
			}(peer, key)
		}
	}
	*resp = *p.respquora.Wait(key)
	return nil
}
