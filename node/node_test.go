package node

import (
	"log"
	"net"
	"testing"
	"time"
)

func TestPeerComms(t *testing.T) {
	log.Println("Testing Comms")
	addr1, err := net.ResolveTCPAddr("tcp", "localhost:5002")
	if err != nil {
		t.Error(err)
	}
	log.Println(addr1, err)
	local1 := &Peer{
		TCPAddr: addr1,
		ID:      ID{false},
	}

	addr2, err := net.ResolveTCPAddr("tcp", "localhost:5004")
	log.Println(addr2, err)
	if err != nil {
		t.Error(err)
	}

	local2 := &Peer{
		TCPAddr: addr2,
		ID:      ID{false},
	}
	go local1.serve()
	go local2.serve()
	err = local2.connect()
	go local1.send(local2, "SUP", "yo")
	go local1.send(local2, "SUP", "yo4")
	go local1.send(local2, "SUP", "yo2")
	go local1.send(local2, "SUP", "yo3")
	go local1.send(local2, "LEAVE", "yo2")
	go local1.send(local2, "MERGE", "yo3")
	go local1.send(local2, "SUP", "yo")
	go local1.send(local2, "SUP", "yo4")
	go local1.send(local2, "SUP", "yo2")
	go local1.send(local2, "SUP", "yo3")
	go local1.send(local2, "LEAVE", "yo2")
	go local1.send(local2, "MERGE", "yo3")
	go local1.send(local2, "SUP", "yo")
	go local1.send(local2, "SUP", "yo4")
	go local1.send(local2, "SUP", "yo2")
	go local1.send(local2, "SUP", "yo3")
	go local1.send(local2, "LEAVE", "yo2")
	go local1.send(local2, "MERGE", "yo3")
	time.Sleep(1 * time.Second)
	local2.conn.Close()
	time.Sleep(3 * time.Second)
	log.Println("Done Testing Comms")
}
