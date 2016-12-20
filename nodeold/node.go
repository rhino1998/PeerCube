package node

import (
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"net"
	"net/rpc"
	"os"
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

	//For Serving
	cluster *Cluster
	rt      []*Cluster
	data    KVStore

	respquora *RespQuorumRegistry

	//For Clienting
	conn *net.TCPConn
}

func (p *Peer) connect() (err error) {
	p.conn, err = net.DialTCP("tcp", nil, p.TCPAddr)
	return err
}

func (p *Peer) serveRPC(addr *net.TCPAddr) {
	server := rpc.NewServer()
	server.RegisterName("RPCNode", p)
	l, e := net.ListenTCP("tcp", addr)
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

func (p *Peer) serve() {
	l, err := net.ListenTCP("tcp", p.TCPAddr)
	if err != nil {
		fmt.Println("Error listening:", err.Error())
		os.Exit(1)
	}
	// Close the listener when the application closes.
	defer l.Close()
	for {
		// Listen for an incoming connection.
		conn, err := l.AcceptTCP()
		if err != nil {
			fmt.Println("Error accepting: ", err.Error())
			os.Exit(1)
		}
		// Handle connections in a new goroutine.
		go p.handle(conn)
	}

}

func (p *Peer) handle(conn *net.TCPConn) {
	dec := gob.NewDecoder(conn)
	var msg []interface{}
	for {
		msg = nil
		err := dec.Decode(&msg)
		if err == io.EOF {
			return
		}
		if err != nil {
			if err.Error() == "extra data in buffer" { //happens often, is fine
				continue
			}
			log.Printf("%v error from %s %v", err, conn.RemoteAddr().String(), msg)
			continue
		}
		if len(msg) == 0 {
			log.Printf("Empty/Bad message from %s", conn.RemoteAddr().String())
			continue
		}
		var method string //Ensures first part of message is string
		switch msg[0].(type) {
		case string:
			method = msg[0].(string)
		default: //Improperly formed message
			log.Printf("Improperly formed message from %v", conn.RemoteAddr().String())
			continue
		}

		switch method {
		case "LOOKUP":

			//Total msg length
			if len(msg) != 4 {
				log.Printf("Invalid argument count. Lookup takes 4 arguments, got %v", len(msg))
				continue
			}

			//Check 1st arg
			switch msg[1].(type) {
			case Key:
			default:
				log.Printf("Invalid Key, %v", msg[1])
				continue
			}

			//Check 2nd Arg
			switch msg[2].(type) {
			case *Peer:
			default:
				log.Printf("Invalid origin, %v", msg[2])
				continue
			}

			//Check 3rd Arg
			switch msg[3].(type) {
			case *Peer:
			default:
				log.Printf("Invalid previous link, %v", msg[3])
				continue
			}

			go p.lookup(msg[1].(Key), msg[2].(*Peer), msg[3].(*Peer))

		case "JOIN":
		case "JOINSPARE":
		case "JOINTEMP":
		case "JOINACK":
		case "LEAVE":
			log.Println("leave")
		case "SPLIT":
			log.Println("split")
		case "MERGE":
			log.Println("merge")
		case "LOOKUPRETURN":

			//Total msg length
			if len(msg) != 4 {
				log.Printf("Invalid argument count. LOOKUPRETURN takes 4 arguments, got %v", len(msg))
				continue
			}

			//Check 1st arg
			switch msg[1].(type) {
			case Key:
			default:
				log.Printf("Invalid Key, %v", msg[1])
				continue
			}

			//Check 2nd Arg
			switch msg[2].(type) {
			case ID:
			default:
				log.Printf("Invalid cluster label, %v", msg[2])
				continue
			}

			//Check 3rd Arg
			switch msg[3].(type) {
			case []byte:
			default:
				log.Printf("Invalid []byte, %v", msg[3])
				continue
			}

			go p.lookupreturn(msg[1].(Key), msg[2].(ID), msg[3].([]byte))

		default:
			log.Printf("Unrecognized method: %s, %s", method, string(msg[1].(string)))
			continue

		}
	}
}
func (p *Peer) send(peer *Peer, msg ...interface{}) error {
	if msg == nil || len(msg) == 0 {
		return nil
	}
	return gob.NewEncoder(peer.conn).Encode(&msg)
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

func (p *Peer) Lookup(key *Key, resp *Resp) error {
	if !p.respquora.Register(*key, (p.cluster.SMin-1)/3+1) {
		*resp = *p.respquora.Wait(*key)
		return nil
	}
	if p.Type != CORE {
		peer := p.cluster.coreRandomPeer()
		p.send(peer, "LOOKUP", *key, p, p)
	}

	c := p.findClosestCluster(key.ID)
	if p.cluster.Label.eq(c.Label) {
		for _, peer := range p.cluster.coreRandomPeers((p.cluster.SMin-1)/3 + 1) {
			p.send(peer, "LOOKUP", *key, p, p)
		}
	}
	*resp = *p.respquora.Wait(*key)
	return nil
}

func (p *Peer) lookup(key Key, origin *Peer, prev *Peer) {

	c := p.findClosestCluster(key.ID)
	if p.cluster.Label.eq(c.Label) {

		//Ensure following operation not done twice
		if p.respquora.Register(key, (p.cluster.SMin-1)/3+1) {
			//If not done before, send to all of coreset
			//Broadcast, not implemented exactly yet, fudging it for now
			for _, peer := range p.cluster.CoreSet {
				p.send(peer, "LOOKUP", key, origin, p)
			}
			resp := p.respquora.Wait(key)
			p.send(prev, "LOOKUPRETURN", resp.Key, resp.Label, resp.Value)
		}

		//TBI: get data and send back after consensus from (sMin-1)/3 +1 peers

	} else {
		p.respquora.Register(key, (p.cluster.SMin-1)/3+1)

		for _, peer := range p.cluster.coreRandomPeers((p.cluster.SMin-1)/3 + 1) {
			p.send(peer, "LOOKUP", key, origin, p)
		}

		resp := p.respquora.Wait(key)
		p.send(prev, "LOOKUPRETURN", resp.Key, resp.Label, resp.Value)

		//Register a struct that blocks until (sMin-1)/3 +1  similar responses are recieved
		p.respquora.Wait(key)
	}
}

func (p *Peer) lookupreturn(key Key, label ID, val []byte) {
	p.respquora.Check(key, &Resp{key, label, val})
}

func (p *Peer) join(q *Peer) {
	c := p.findClosestCluster(q.ID)
	if p.cluster.Label.eq(c.Label) {
		if p.cluster.Label.prefix(q.ID) {
			//still improper broadcast
			for _, peer := range p.cluster.CoreSet {
				p.send(peer, "JOINSPARE", c, q)
			}
		} else {
			//still improper broadcast
			for _, peer := range p.cluster.CoreSet {
				p.send(peer, "JOINTEMP", c, q)
			}
		}
	} else {
		for _, peer := range p.cluster.coreRandomPeers((p.cluster.SMin-1)/3 + 1) {
			p.send(peer, "JOIN", q)
		}
	}
}

func (p *Peer) joinspare(c *Cluster, q *Peer) {

}

func splitRoutingTable() {

}
