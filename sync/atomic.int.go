package chansync

/*
* CODE GENERATED AUTOMATICALLY WITH github.com/firelizzard18/go-misc/sync/gen
* THIS FILE SHOULD NOT BE EDITED BY HAND
*/

// AtomicInt is a concurrency-safe, atomic int.
type AtomicInt interface {
	// Read returns the internal int value.
	Read() int
	// Write sets the internal int value to val, if and only if the current
	// interal value matches old. Write returns whether or not the write was
	// successful.
	Write(old, val int) bool
}

type atomicInt struct {
	read chan int
	write chan *atomicIntWrite
}

type atomicIntWrite struct {
	old, val int
	ret chan bool
}

// NewAtomicInt returns a new atomic int.
func NewAtomicInt(val int) AtomicInt {
	a := &atomicInt {
		read: make(chan int),
		write: make(chan *atomicIntWrite),
	}

	go func() {
		for {
			last := val
			select {
			case a.read <- val:
				// nothing else to do
			case wr := <- a.write:
				if wr.old != last {
					wr.ret <- false
				} else {
					val = wr.val
					wr.ret <- true
				}
			}
		}
	}()

	return a
}

func (a *atomicInt) Read() int {
	return <- a.read
}

func (a *atomicInt) Write(old, val int) bool {
	ret := make(chan bool)
	a.write <- &atomicIntWrite{old: old, val: val, ret: ret}
	return <- ret
}
