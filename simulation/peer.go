package simulation

type PeerType byte

type Peer interface {
	GetType() PeerType
	SetType(PeerType)
	ID() ID
	Join(*Cluster) *Cluster
	Leave()
	GetCluster() *Cluster
	SetCluster(*Cluster)
	SetClusterType(*Cluster, PeerType)

	Put(ID, string) (string, error)
	Get(ID) (string, error)
	Delete(ID) (string, error)

	put(ID, string) bool
	get(ID) (string, bool)
	delete(ID) (string, bool)
	dataStore() DataStore
	replaceData(DataStore)

	flushPutWait(ID, string, error)
	addPutWait(ID) (chan string, chan error)

	flushPutReqs(ID) map[string]Peer
	addPutReq(ID, Peer)

	flushLookupWait(ID, string, error)
	addLookupWait(ID) (chan string, chan error)

	flushLookupReqs(ID) map[string]Peer
	addLookupReq(ID, Peer)

	flushDeleteWait(ID, string, error)
	addDeleteWait(ID) (chan string, chan error)

	flushDeleteReqs(ID) map[string]Peer
	addDeleteReq(ID, Peer)

	send(msg Message)
	serve()
	ready()
	splitlock()
	unsplitlock()
	rsplitlock()
	runsplitlock()
	lock()
	unlock()
}
