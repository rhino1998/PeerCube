package node

import (
	"fmt"
	"log"
	"testing"
	"time"
)

func TestWait(t *testing.T) {
	log.Println("Testing Wait")
	w := NewWait()
	go func() {
		fmt.Println(w.Wait())
	}()
	go func() {
		fmt.Println(w.Wait())
	}()
	go func() {
		fmt.Println(w.Wait())
	}()
	go func() {
		fmt.Println(w.Wait())
	}()
	go func() {
		fmt.Println(w.Wait())
	}()
	time.Sleep(2 * time.Second)
	w.Close(&Resp{ID{}, ID{}, []byte("yo")})
	fmt.Println(w)
}
