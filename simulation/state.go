package simulation

type State struct {
	Cluster *Cluster
	Data    map[string][]byte
	Type    PeerType
	Rt      []*Cluster
	Pt      map[string]*Cluster
}
