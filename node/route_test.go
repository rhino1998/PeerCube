package node

import "testing"

func TestDistance(t *testing.T) {
	a := ID{true, false, true, false, true, false, true, true, false, true}
	b := ID{true, false, true, false, true}

	d := distance(a, b)
	if d.String() != "0000001101" {
		t.Errorf("%s^%s!=%s", a.String(), b.String(), d.String())
	}

}
