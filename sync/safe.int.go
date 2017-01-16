package chansync

/*
* CODE GENERATED AUTOMATICALLY WITH github.com/firelizzard18/go-misc/sync/gen
* THIS FILE SHOULD NOT BE EDITED BY HAND
*/

// SafeInt is a concurrency-safe int.
type SafeInt interface {
	// Read returns the internal int value.
	Read() int
	// Write sets the internal int value to val and returns the previous
	// value.
	Write(val int) int
}

type safeInt struct {
	read chan int
	write chan *safeIntWrite
}

type safeIntWrite struct {
	val int
	ret chan int
}

// NewSafeInt returns a new safe int.
func NewSafeInt(val int) SafeInt {
	s := &safeInt {
		read: make(chan int),
		write: make(chan *safeIntWrite),
	}

	go func() {
		for {
			last := val
			select {
			case s.read <- val:
				// nothing else to do
			case wr := <- s.write:
				val = wr.val
				wr.ret <- last
			}
		}
	}()

	return s
}

func (s *safeInt) Read() int {
	return <- s.read
}

func (s *safeInt) Write(val int) int {
	ret := make(chan int)
	s.write <- &safeIntWrite{val: val, ret: ret}
	return <- ret
}
