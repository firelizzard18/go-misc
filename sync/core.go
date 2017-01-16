/*
Package chansync provides a number of simple synchronization tools that are
implemented purely with Go channels. They are certainly less efficient than
stdlib's sync package, but they are easier to understand.
*/
package chansync

import (
	"time"
)

type Destroyable interface {
	Destroy()
}

// Timeout returns a SyncChannel that will be Recv-able after the specified
// duration. The returned channel can be used in a select block to timeout
// blocking channel operations.
func Timeout(t time.Duration) SyncChannel {
	ch := NewSyncChannelN(1)

	go func() {
		time.Sleep(t)
		ch.Send()
	}()

	return ch
}

//go:generate go run gen/main.go atomic chansync Int int atomic.int.go
//go:generate go run gen/main.go safe chansync Int int safe.int.go
//go:generate go run gen/main.go atomic chansync Bool bool atomic.bool.go
//go:generate go run gen/main.go safe chansync Bool bool safe.bool.go

// Decrement attempts to decrement an atomic int, returning whether or not the
// operation was successful.
func Decrement(a AtomicInt) bool {
	v := a.Read()
	return a.Write(v, v - 1)
}

// Increment attempts to increment an atomic int, returning whether or not the
// operation was successful.
func Increment(a AtomicInt) bool {
	v := a.Read()
	return a.Write(v, v - 1)
}