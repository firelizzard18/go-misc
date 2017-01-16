package chansync

import (
	"time"
)


// Zero-length constant
var empty = struct{}{}

// ChannelOpResult represents the result of a channel operation
type ChannelOpResult uint8

const (
	// ChannelOpSuccess indicates a successful channel operation
	ChannelOpSuccess ChannelOpResult = iota

	// ChannelOpFailure indicates an unsuccessful channel operation
	ChannelOpFailure

	// ChannelOpTimeout indicates that a channel operation timed out
	ChannelOpTimeout

	// ChannelOpClosed indicates that a channel operation failed due to the channel closing
	ChannelOpClosed
)

/*
SyncChannel is a light wrapper around a channel intended only for
synchronization purposes. It cannot be used to transmit data.
*/
type SyncChannel chan struct{}

// NewSyncChannel returns an unbuffered SyncChannel
func NewSyncChannel() SyncChannel {
	return SyncChannel(make(chan struct{}))
}

// NewSyncChannelN returns a SyncChannel with the specified buffer depth
func NewSyncChannelN(n int) SyncChannel {
	return SyncChannel(make(chan struct{}, n))
}

// Send writes a signal to the channel. If the channel cannot be written to,
// Send blocks until it can. If the channel is closed while Send is blocking,
// Send panics.
func (ch SyncChannel) Send() {
	ch <- empty
}

// TrySend attempts to write a signal to the channel. If the channel cannot be
// written to, TrySend returns ChannelOpFailure. Otherwise, TrySend returns
// ChannelOpSuccess. If the channel is closed, TrySend panics.
func (ch SyncChannel) TrySend() ChannelOpResult {
	select {
	case ch <- empty:
		return ChannelOpSuccess
	default:
		return ChannelOpFailure
	}
}

// Recv reads a signal from the channel. If the channel cannot be read from,
// Recv blocks until it can. If the channel is closed, TrySend returns
// ChannelOpClosed. Otherwise, Recv returns ChannelOpSuccess.
func (ch SyncChannel) Recv() ChannelOpResult {
	_, ok := <- ch
	if ok {
		return ChannelOpSuccess
	}
	return ChannelOpClosed
}

// TryRecv attempts to read a signal from the channel. If the channel cannot be
// read from TryRecv returns ChannelOpFailure. If the channel is closed,
// TryRecv returns ChannelOpClosed. Otherwise, TryRecv returns
// ChannelOpSuccess.
func (ch SyncChannel) TryRecv() ChannelOpResult {
	select {
	case _, ok := <- ch:
		if ok {
			return ChannelOpSuccess
		}
		return ChannelOpClosed
	default:
		return ChannelOpFailure
	}
}

// TimeoutRecv attempts to read a signal from the channel, with a timeout. If
// the channel cannot be read from before the timeout expires, TimeoutRecv
// returns ChannelOpTimeout. If the channel is closed, TimeoutRecv returns
// ChannelOpClosed. Otherwise, TimeoutRecv returns ChannelOpSuccess.
func (ch SyncChannel) TimeoutRecv(timeout time.Duration) ChannelOpResult {
	select {
	case _, ok := <- ch:
		if ok {
			return ChannelOpSuccess
		}
		return ChannelOpClosed
	case <- Timeout(timeout):
		return ChannelOpTimeout
	}
}

// Close closes the underlying chan that SyncChannel uses. This will result
// in any currently blocked send calls or future send calls panicing, and 
// any currently blocked receive calls or future recieve calls returning
// ChannelOpClosed.
func (ch SyncChannel) Close() {
	close(ch)
}