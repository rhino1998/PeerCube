package node

import (
	"math/rand"
)

type ID []bool

func distance(a, b ID) ID {
	a, b = equalize_length(a, b)
	sum := make(ID, len(a))
	for i := 0; i < len(sum); i++ {
		sum[i] = (a[i] != b[i])
	}
	return sum
}

func hamming_distance(a, b ID) uint64 {
	a, b = equalize_length(a, b)
	dist := uint64(0)
	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			dist++
		}
	}
	return dist
}

func IDFromUint64(val uint64) ID {
	id := make(ID, 64)
	for i := uint64(0); i < 64; i++ {
		id[i] = (val<<(63-i))>>i == 1
	}
	return id
}

func IDFromString(val string) ID {
	id := make(ID, len(val))
	for i, char := range val {
		id[i] = char == '1'
	}
	return id
}

func RandomID(length uint64) ID {
	id := make(ID, length)
	for i := uint64(0); i < length; i++ {
		id[i] = rand.Intn(2) == 1
	}
	return id
}

func equalize_length(a, b ID) (ID, ID) {
	if len(a) < len(b) {
		a = append(a, make(ID, len(b)-len(a))...)
	}
	if len(b) < len(a) {
		b = append(b, make(ID, len(a)-len(b))...)
	}
	return a, b
}

func (id ID) rshift(amt uint64) ID {
	r := make(ID, amt)
	return append(r, id...)[:len(id)]
}
func (id ID) lshift(amt uint64) ID {
	r := make(ID, amt)
	return append(id[amt:], r...)
}

func (id ID) prefix(other ID) bool {
	if len(id) > len(other) {
		return false
	}
	for i := 0; i < len(id); i++ {
		if id[i] != other[i] {
			return false
		}
	}
	return true
}

func (id ID) lt(other ID) bool {
	id, other = equalize_length(id, other)
	for i := 0; i < len(id); i++ {
		if id[i] != other[i] {
			return other[i]
		}
	}
	return false
}

func (id ID) lte(other ID) bool {
	id, other = equalize_length(id, other)
	for i := 0; i < len(id); i++ {
		if id[i] != other[i] {
			return other[i]
		}
	}
	return false
}

func (id ID) gt(other ID) bool {
	id, other = equalize_length(id, other)
	for i := 0; i < len(id); i++ {
		if id[i] != other[i] {
			return id[i]
		}
	}
	return false
}

func (id ID) gte(other ID) bool {
	id, other = equalize_length(id, other)
	for i := 0; i < len(id); i++ {
		if id[i] != other[i] {
			return id[i]
		}
	}
	return true
}

func (id ID) eq(other ID) bool {
	id, other = equalize_length(id, other)
	for i := 0; i < len(id); i++ {
		if id[i] != other[i] {
			return false
		}
	}
	return true
}
func (id ID) String() string {
	str := make([]rune, len(id))
	for i := 0; i < len(id); i++ {
		if id[i] {
			str[i] = '0'
		} else {
			str[i] = '1'
		}
	}
	return string(str)
}
