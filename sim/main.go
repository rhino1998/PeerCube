package main

import (
	"fmt"
	"os"
	"sync/atomic"
	"time"

	"github.com/rhino1998/peercube/simulation"
)

func main() {

	totalRPS := float64(0)
	eachMinRPS := float64(0)
	minRPS := float64(0)
	requests := uint64(0)
	latency := uint64(0)
	latencyAvg := float64(0)
	eachMinAvgLatency := float64(0)
	totalAvgLatency := float64(0)

	for simulation.PeerRegistry.Length() < 512 {
		p := simulation.NewStdPeer()
		go p.Join(simulation.BootstrapCluster)
		time.Sleep(80 * time.Millisecond)
	}
	time.Sleep(8 * time.Second)
	simulation.Mu = 0.0
	go func() {
		max := float64(6)
		for c := 0; c < int(max); c++ {
			duration := 6 * time.Second
			select {
			case <-time.After(duration):
				lat := atomic.SwapUint64(&latency, 0)
				req := atomic.SwapUint64(&requests, 0)
				minRPS = float64(req) / float64(duration/time.Second)
				latencyAvg = float64(lat) / float64(req)
				eachMinRPS += minRPS
				eachMinAvgLatency += latencyAvg
				fmt.Println("ALERTALERT   --------------------   AVG RPS: ", minRPS)
				fmt.Println("ALERTALERT   --------------------   AVG LATENCY: ", time.Duration(latencyAvg))
				fmt.Println(c)
			}
		}
		totalRPS = eachMinRPS / max
		totalAvgLatency = eachMinAvgLatency / max

		fmt.Println("RPS: ", totalRPS)
		fmt.Println("LATENCY: ", time.Duration(totalAvgLatency))
		os.Exit(0)
	}()

	sema := make(chan struct{}, 64)
	for simulation.PeerRegistry.Length() > simulation.TSplit {
		//ClusterRegistry.PrintAll()
		//fmt.Println(simulation.PeerRegistry.Length(), ":", simulation.ClusterRegistry.SizeAll())

		var p simulation.Peer
		for _, fp := range simulation.PeerRegistry.Peers() {
			if fp.GetType() == simulation.CORE {
				p = fp
				break
			}
		}
		if p != nil {
			go func() {
				sema <- struct{}{}
				defer func() { <-sema }()
				key := simulation.RandomID(simulation.M)
				start := time.Now()
				_, err := p.Put(key, simulation.RandomID(simulation.M).String())
				atomic.AddUint64(&latency, uint64(time.Since(start)))
				if err != nil {
					return
				}
				atomic.AddUint64(&requests, 1)
				time.Sleep(800 * time.Millisecond)
				start = time.Now()
				_, err = p.Get(key)
				atomic.AddUint64(&latency, uint64(time.Since(start)))
				atomic.AddUint64(&requests, 1)
				if err != nil {
					//fmt.Println("FAIL", err)
				} else {
					//fmt.Println("SUCCESSSSSS")
				}
			}()

		}
		if simulation.PeerRegistry.Length() > 200056 {
			for _, fp := range simulation.PeerRegistry.Peers() {
				if fp.GetType() == simulation.CORE {
					fp.Leave()
					break
				}
			}

		}
		time.Sleep(10 * time.Millisecond)
	}
}
