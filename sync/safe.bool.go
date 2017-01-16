package chansync

/*
* CODE GENERATED AUTOMATICALLY WITH github.com/firelizzard18/go-misc/sync/gen
* THIS FILE SHOULD NOT BE EDITED BY HAND
*/

// SafeBool is a concurrency-safe bool.
type SafeBool interface {
	// Read returns the internal bool value.
	Read() bool
	// Write sets the internal bool value to val and returns the previous
	// value.
	Write(val bool) bool
}

type safeBool struct {
	read chan bool
	write chan *safeBoolWrite
}

type safeBoolWrite struct {
	val bool
	ret chan bool
}

// NewSafeBool returns a new safe bool.
func NewSafeBool(val bool) SafeBool {
	s := &safeBool {
		read: make(chan bool),
		write: make(chan *safeBoolWrite),
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

func (s *safeBool) Read() bool {
	return <- s.read
}

func (s *safeBool) Write(val bool) bool {
	ret := make(chan bool)
	s.write <- &safeBoolWrite{val: val, ret: ret}
	return <- ret
}
