package node

type Wait struct {
	c chan *Resp
}

func NewWait() *Wait {
	return &Wait{
		c: make(chan *Resp),
	}
}

func (w *Wait) Close(resp *Resp) {
	for {
		select {
		case w.c <- resp:
			continue
		default:
			return
		}
	}
}

func (w *Wait) Wait() *Resp {
	r := <-w.c
	return r
}
