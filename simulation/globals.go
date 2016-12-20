package simulation

import "time"

const (
	M           int           = 256
	Mu          float64       = 0
	SMax        int           = 32
	SMin        int           = 8
	SafteyValue int           = ((SMin-1)/3 + 1)
	TSplit      int           = 16
	Timeout     time.Duration = 10000 * time.Millisecond
	QueueSize   int           = 256
	WorkerCount int           = 8
	Latency     time.Duration = 10 * time.Millisecond
)

//Fako enum thingy
const (
	UNDEF PeerType = iota
	CORE
	TEMP
	SPARE
)

var (
	ClusterRegistry  *clusterRegistry
	PeerRegistry     *peerRegistry
	BootstrapCluster *Cluster
)

func init() {
	ClusterRegistry = NewClusterRegistry()
	PeerRegistry = NewPeerRegistry()

	BootstrapCluster = NewCluster()
	for i := 0; i < SMin; i++ {
		BootstrapCluster.addVc(NewStdPeer())
	}
	for _, p := range BootstrapCluster.vc {
		p.serve()
	}
}
