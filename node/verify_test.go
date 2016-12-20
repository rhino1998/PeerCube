package node

import "testing"

func TestRespQuorum(t *testing.T) {
	v := NewRespQuorum(8)
	go func() {
		for i := 0; i < 8; i++ {
			go v.Check(&Resp{Key: ID{}, Label: ID{}, Value: []byte("onething")})
		}
	}()
	go func() {
		for i := 0; i < 6; i++ {
			go v.Check(&Resp{Key: ID{}, Label: ID{}, Value: []byte("d")})
		}
	}()
	if q := v.Wait(); string(q.Value) != "onething" {
		t.Error(q)
	}
}
