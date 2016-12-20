package simulation

import (
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"
)

func TestSimulation(t *testing.T) {
	runtime.LockOSThread()

	wg := &sync.WaitGroup{}
	d := func(v int) {
		for i := 0; i < v; i++ {
			wg.Add(1)
			go func(i int) {
				p := NewStdPeer()
				p.Join(BootstrapCluster)
				//fmt.Println(i, p.GetCluster().Label().String())
				wg.Done()
			}(i)
			time.Sleep(200 * time.Millisecond)
		}
	}
	go d(5000)
	time.Sleep(4 * time.Second)
	for PeerRegistry.Length() > TSplit {
		//ClusterRegistry.PrintAll()
		fmt.Println(PeerRegistry.Length(), ":", ClusterRegistry.SizeAll())
		fmt.Println("SIZEALL")
		if true {
			fmt.Println("DoneSIZEALL")
			var p Peer
			PeerRegistry.mutex.RLock()
			for _, fp := range PeerRegistry.peers {
				if fp.GetType() == CORE {
					p = fp
					break
				}
			}
			PeerRegistry.mutex.RUnlock()
			if p != nil {
				key := RandomID(M)
				data, err := p.Put(key, RandomID(M).String())
				fmt.Println(data, err)
				data, err = p.Get(key)
				fmt.Println(data, err)
			}

		}
		time.Sleep(100 * time.Millisecond)
	}
}
