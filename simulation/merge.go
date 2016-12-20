package simulation

import (
	"fmt"
	"sync/atomic"
)

func Merge(a, b *Cluster) {
	if atomic.SwapUint64(&b.merge, 1) == 1 {
		return
	}
	b.mutex.Lock()

	fmt.Println("MERGING", a.label.String(), "  ", b.label.String())
	label := make(ID, len(a.label)-1, M)
	copy(label, a.label)
	mergedCluster := &Cluster{
		label:      label,
		vc:         make(map[string]Peer),
		vs:         make(map[string]Peer),
		vt:         make(map[string]Peer),
		rt:         make([]*Cluster, len(a.label)-1, M),
		addHandler: NewPeerHandler(SafteyValue),
	}
	copy(mergedCluster.rt, a.rt)
	i := 0
	copyPeer := make([]Peer, 2)
	for j, cluster := range []*Cluster{a, b} {
		for _, peer := range cluster.vc {
			copyPeer[j] = peer
			break
		}
	}

	for j, cluster := range []*Cluster{a, b} {
		for id, peer := range cluster.vc {
			if i < SMin {
				mergedCluster.vc[id] = peer
				peer.dataStore().Merge(copyPeer[1-j].dataStore())
				peer.SetClusterType(mergedCluster, CORE)
				i++
			} else {
				mergedCluster.vs[id] = peer
				peer.SetClusterType(mergedCluster, SPARE)
				peer.replaceData(&MapWrapper{data: make(map[string]string)})
			}
		}
	}
	ClusterRegistry.Merge(a, b, mergedCluster)
}
