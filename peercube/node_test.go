package peercube

import (
	"fmt"
	"log"
	"testing"
	"time"
)

func TestPeerComms(t *testing.T) {
	//runtime.GOMAXPROCS(1)
	log.Println("Testing Comms")
	for i := uint64(0); i < SMin-1; i++ {
		NewPeer().Type = CORE
	}
	local1 := NewPeer()
	local1.Type = CORE
	for _, peer := range peerregistry.data {
		for cID, contact := range peerregistry.data {
			peer.cluster.Vc[cID] = contact
		}
		go peer.serve()
	}
	local2 := NewPeer()
	log.Println(local2.joinByID(local1.ID))
	local2.mutex.RLock()
	fmt.Println(local2.Type)
	local2.mutex.RUnlock()

	for i := uint64(0); i < 29; i++ {
		fmt.Println("Adding", len(peerregistry.data))
		p := NewPeer()
		p.joinByID(local1.ID)
		time.Sleep(200 * time.Millisecond)
		p.mutex.RLock()
		fmt.Println("Added", peerregistry.Length()-1, "to", p.cluster.Label.String())
		p.mutex.RUnlock()
	}
	time.Sleep(2 * time.Second)
	for _, apeer := range peerregistry.data {
		apeer.mutex.RLock()
		if apeer.Type != CORE {
			continue
		}
		fmt.Printf("%s | %02d | %03d | %s |RT:[", apeer.ID.String(), apeer.Type, apeer.N, apeer.cluster.Label.String())
		for _, c := range apeer.rt {
			fmt.Printf("%2s, ", c.Label.String())
		}
		fmt.Print("] PT: [")
		for _, c := range apeer.pt {
			fmt.Printf("%2s, ", c.Label.String())
		}
		fmt.Println("]")
		apeer.mutex.RUnlock()
	}

	log.Println("Done Testing Comms")
}
