package simulation

import (
	"fmt"
	"sync"
)

type ClusterExistsError struct {
	label ID
}

func NewClusterExistsError(label ID) *ClusterExistsError {
	return &ClusterExistsError{label: label}
}

func (e ClusterExistsError) Error() string {
	return fmt.Sprintf("Cluster %s already exists", e.label.String())
}

type clusterRegistry struct {
	mutex    sync.RWMutex
	clusters map[string]*Cluster
}

func NewClusterRegistry() *clusterRegistry {
	return &clusterRegistry{clusters: make(map[string]*Cluster)}
}

func (cr *clusterRegistry) Get(label ID) *Cluster {
	cr.mutex.RLock()
	defer cr.mutex.RUnlock()
	return cr.clusters[label.String()]
}

func (cr *clusterRegistry) Delete(label ID) {
	cr.mutex.Lock()
	defer cr.mutex.Unlock()
	delete(cr.clusters, label.String())
}

func (cr *clusterRegistry) Update(newlabel ID, c *Cluster) {
	cr.mutex.Lock()
	defer cr.mutex.Unlock()
	delete(cr.clusters, c.label.String())
	cr.clusters[newlabel.String()] = c
}

func (cr *clusterRegistry) Merge(a, b, c *Cluster) {
	cr.mutex.Lock()
	defer cr.mutex.Unlock()
	delete(cr.clusters, a.label.String())
	delete(cr.clusters, b.label.String())
	cr.clusters[c.label.String()] = c
}

func (cr *clusterRegistry) PrintAll() {
	cr.mutex.RLock()
	defer cr.mutex.RUnlock()

	for label, _ := range cr.clusters {
		fmt.Print(label, ",")
	}
	fmt.Println()
}

func (cr *clusterRegistry) SizeAll() int {
	cr.mutex.RLock()
	defer cr.mutex.RUnlock()

	sum := 0
	for _, c := range cr.clusters {
		sum += len(c.vc) + len(c.vs)
	}
	return sum
}

func (cr *clusterRegistry) Register(cluster *Cluster) error {
	cr.mutex.Lock()
	defer cr.mutex.Unlock()

	if val, found := cr.clusters[cluster.label.String()]; found && val != cluster {
		return NewClusterExistsError(cluster.label)
	}
	cr.clusters[cluster.label.String()] = cluster
	return nil
}

func (cr *clusterRegistry) FixAllRT() {
	cr.mutex.RLock()
	defer cr.mutex.RUnlock()
	for _, c := range cr.clusters {
		var min *Cluster
		for i, _ := range c.rt {
			ideal := make(ID, len(c.label), M)
			copy(ideal, c.label)
			ideal[i] = !ideal[i]
			for _, rc := range cr.clusters {
				if rc != c && (min == nil || distance(min.label, ideal).gt(distance(rc.label, ideal))) {
					min = rc
				}
			}
			c.rt[i] = min
		}
	}

}

type PeerExistsError struct {
	id ID
}

func NewPeerExistsError(id ID) *PeerExistsError {
	return &PeerExistsError{id: id}
}

func (e PeerExistsError) Error() string {
	return fmt.Sprintf("Peer %s already exists", e.id.String())
}

type peerRegistry struct {
	mutex sync.RWMutex
	peers map[string]Peer
}

func NewPeerRegistry() *peerRegistry {
	return &peerRegistry{peers: make(map[string]Peer)}
}

func (pr *peerRegistry) Get(id ID) Peer {
	pr.mutex.RLock()
	defer pr.mutex.RUnlock()
	return pr.peers[id.String()]
}

func (pr *peerRegistry) Delete(id ID) {
	pr.mutex.RLock()
	defer pr.mutex.RUnlock()
	delete(pr.peers, id.String())
}

func (pr *peerRegistry) Length() int {
	pr.mutex.RLock()
	defer pr.mutex.RUnlock()
	return len(pr.peers)
}

func (pr *peerRegistry) Register(peer Peer) error {
	pr.mutex.Lock()
	defer pr.mutex.Unlock()

	if val, found := pr.peers[peer.ID().String()]; found && val != peer {
		return NewPeerExistsError(peer.ID())
	}
	pr.peers[peer.ID().String()] = peer
	return nil
}

func (pr *peerRegistry) Peers() map[string]Peer {
	pr.mutex.RLock()
	pr2 := make(map[string]Peer)
	for id, peer := range pr.peers {
		pr2[id] = peer
	}
	pr.mutex.RUnlock()
	return pr2
}
