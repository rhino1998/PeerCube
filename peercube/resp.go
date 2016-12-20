package peercube

type Resp struct {
	Key   ID
	Label ID
	Value []byte
}

func (r *Resp) equals(o *Resp) bool {
	return r.Key.eq(o.Key) && r.Label.eq(o.Label) && string(r.Value) == string(o.Value)

}

type JoinAck struct {
	Type    byte
	Cluster *Cluster
}

func (a *JoinAck) equals(o *JoinAck) bool {
	return a.Type == o.Type && a.Cluster.Label.eq(o.Cluster.Label)
}
