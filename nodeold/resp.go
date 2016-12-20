package node

type Resp struct {
	Key   Key
	Label ID
	Value []byte
}

func (r *Resp) equals(o *Resp) bool {
	return r.Key.Key == o.Key.Key && r.Label.eq(o.Label) && string(r.Value) == string(o.Value)
}
