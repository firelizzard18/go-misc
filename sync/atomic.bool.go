package chansync

/*
* CODE GENERATED AUTOMATICALLY WITH github.com/firelizzard18/go-misc/sync/gen
* THIS FILE SHOULD NOT BE EDITED BY HAND
*/

// AtomicBool is a concurrency-safe, atomic bool.
type AtomicBool interface {
	// Read returns the internal bool value.
	Read() bool
	// Write sets the internal bool value to val, if and only if the current
	// interal value matches old. Write returns whether or not the write was
	// successful.
	Write(old, val bool) bool
}

type atomicBool struct {
	read chan bool
	write chan *atomicBoolWrite
}

type atomicBoolWrite struct {
	old, val bool
	ret chan bool
}

// NewAtomicBool returns a new atomic bool.
func NewAtomicBool(val bool) AtomicBool {
	a := &atomicBool {
		read: make(chan bool),
		write: make(chan *atomicBoolWrite),
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

func (a *atomicBool) Read() bool {
	return <- a.read
}

func (a *atomicBool) Write(old, val bool) bool {
	ret := make(chan bool)
	a.write <- &atomicBoolWrite{old: old, val: val, ret: ret}
	return <- ret
}
