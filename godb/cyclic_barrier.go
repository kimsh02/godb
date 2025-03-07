package godb

import (
	"sync"
)

// See: https://medium.com/golangspec/reusable-barriers-in-golang-156db1f75d0b

type Barrier struct {
	c      int
	n      int
	m      sync.Mutex
	before chan int
	after  chan int
}

func NewBarrier(n int) *Barrier {
	b := Barrier{
		n:      n,
		before: make(chan int, 1),
		after:  make(chan int, 1),
	}
	b.after <- 1
	return &b
}

func (b *Barrier) Wait() {
	b.m.Lock()
	b.c += 1
	if b.c == b.n {
		// close 2nd gate
		<-b.after
		// open 1st gate
		b.before <- 1
	}
	b.m.Unlock()

	// pass through 1st gate and leave open
	<-b.before
	b.before <- 1

	b.m.Lock()
	b.c -= 1
	if b.c == 0 {
		// close 1st gate
		<-b.before
		// open 2st gate
		b.after <- 1
	}
	b.m.Unlock()

	<-b.after
	b.after <- 1
}

func (b *Barrier) Done() {
	b.m.Lock()
	b.n -= 1
	if b.c >= b.n {
		// close 2nd gate
		<-b.after
		// open 1st gate
		b.before <- 1
	}
	b.m.Unlock()
}
