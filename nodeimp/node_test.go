package node

import (
	"log"
	"testing"
	"time"
)

func TestPeerComms(t *testing.T) {
	log.Println("Testing Comms")
	local1, err := NewServerNode("localhost:5000", 128, 8, 32, 16)
	local2, err := NewServerNode("localhost:5001", 128, 8, 32, 16)
	log.Println(err)
	go local1.serve()
	go local2.serve()
	time.Sleep(1 * time.Second)
	log.Println(local2.joinByAddr("localhost:5000"))
	log.Println()
	time.Sleep(3 * time.Second)
	log.Println("Done Testing Comms")
}
