package chansync

import (
	"time"
)

// Semaphore is a semaphore driven by channels. Nothing special here.
type Semaphore interface {
	// Acquire blocks until the specified number of resources are obtained.
	Acquire(n int)
	// TryAcquire attempts to acquire the specified number of resources. If
	// the operation fails, TryAcquire returns false. Otherwise, TryAcquire
	// returns true.
	TryAcquire(n int) bool
	// Release releases the specified number of resources.
	Release(n int)
}

type semaphore struct {
	size int

	count AtomicInt
	event Event
}

// NewSemaphore returns a new Semaphore with the specified number of total/max
// resources and the specified number of starting resources.
func NewSemaphore(size, start int) Semaphore {
	return &semaphore {
		size: size,

		count: NewAtomicInt(start),
		event: NewEvent(),
	}
}

func (s *semaphore) Acquire(n int) {
	for {
		v := s.count.Read()
		if n > v {
			s.event.TrySubscribe(time.Microsecond)
		} else if s.count.Write(v, v - n) {
			return
		}
	}
}

func (s *semaphore) TryAcquire(n int) bool {
	v := s.count.Read()

	if !s.count.Write(v, v - n) {
		return false
	}

	return true
}

func (s *semaphore) Release(n int) {
	for {
		var next int

		// get the current count
		v := s.count.Read()

		if n < 0 || n + v > s.size {
			next = s.size
		} else {
			next = n + v
		}

		// 
		if !s.count.Write(v, next) {
			continue
		}

		s.event.PublishAll()
	}
}