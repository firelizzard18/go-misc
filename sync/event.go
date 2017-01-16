package chansync

import (
	"time"
)

// Event is a one-to-many synchronization tool.
type Event interface {
	Destroyable
	// PublishOne unblocks the oldest call to Subscribe. PublishOne is a noop
	// if there are no blocked calls to Subscribe.
	PublishOne()
	// PublishAll unblocks every call to Subscribe. PublishAll is a noop if
	// there are no blocked calls to Subscribe.
	PublishAll()
	// Subscribe will block until unblocked by PublishOne or PublishAll. If
	// the event is destroyed before publish is called, Subscribe returns
	// ChannelOpClosed. Otherwise Subscribe returns ChannelOpSuccess.
	Subscribe() ChannelOpResult
	// TrySubscribe will block until unblocked by PublishOne or PublishAll,
	// with a timeout. If the timeout expires, TrySubscribe returns
	// ChannelOpTimeout. If the event is destroyed before publish is called,
	// TrySubscribe returns ChannelOpClosed. Otherwise, TrySubscribe returns
	// ChannelOpSuccess.
	TrySubscribe(timeout time.Duration) ChannelOpResult
}


type event struct {
	subs []SyncChannel

	destroy SyncChannel
	publish chan bool
	newsubs chan SyncChannel
}

// NewEvent returns a new event.
func NewEvent() Event {
	e := &event{
		subs: make([]SyncChannel, 0, 5),

		destroy: NewSyncChannel(),
		publish: make(chan bool, 1),
		newsubs: make(chan SyncChannel),
	}

	go func() {
		for {
			select {
			case <- e.destroy:
				e.destroy.Close()
				close(e.publish)
				close(e.newsubs)
				for _, sub := range e.subs {
					sub.Close()
				}
				break

			case sub := <- e.newsubs:
				e.subs = append(e.subs, sub)

			case all := <- e.publish:
				if (all) {
					subs := e.subs
					e.subs = make([]SyncChannel, 0, 5)
					go func() {
						for _, sub := range subs {
							sub.Send()
						}
					}()
				} else {
					if len(e.subs) == 0 {
						continue
					}
					sub := e.subs[0]
					e.subs = e.subs[1:]
					go func() { sub.Send() }()
				}
			}
		}
	}()

	return e
}

func (e *event) Destroy() {
	e.destroy.Send()
}

func (e *event) PublishOne() {
	e.publish <- false
}

func (e *event) PublishAll() {
	e.publish <- true
}

func (e *event) newSub() SyncChannel {
	sub := NewSyncChannelN(1)
	go func() { e.newsubs <- sub }()
	return sub
}

func (e *event) Subscribe() ChannelOpResult {
	return e.newSub().Recv()
}

func (e *event) TrySubscribe(timeout time.Duration) ChannelOpResult {
	return e.newSub().TimeoutRecv(timeout)
}