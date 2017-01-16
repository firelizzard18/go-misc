package chansync

// Lock is a simple lock driven by channels. Nothing special here. Lock
// acquisition is first-come, first-served.
type Lock interface {
	// Acquire will block until the lock can be acquired. Acquire returns a
	// reference that can be used to release the lock.
	Acquire() Unlock
	// TryAcquire attempts to acquire the lock. If the lock cannot be acquired,
	// TryAcquire returns (nil, false). If the lock is acquired, TryAcquire
	// returns (u, bool) where u is a reference that can be used to release the
	// lock.
	TryAcquire() (Unlock, bool)
}

// Unlock is a reference that can be used to release a lock.
type Unlock interface {
	// Release releases the lock.
	Release()
}

type lock struct {
	ch SyncChannel
}

type unlock struct {
	ch SyncChannel
}

// NewLock returns a new Lock
func NewLock() Lock {
	return &lock{
		ch: NewSyncChannelN(1),
	}
}

func (l *lock) Acquire() Unlock {
	l.ch.Send()
	u := l.newUnlock()
	return u
}

func (l *lock) TryAcquire() (Unlock, bool) {
	if l.ch.TrySend() == ChannelOpSuccess {
		return l.newUnlock(), true
	} else {
		return nil, false
	}
}

func (l *lock) newUnlock() Unlock {
	u := &unlock {
		ch: NewSyncChannel(),
	}
	go func() {
		u.ch.Recv()
		l.ch.Recv()
	}()
	return u
}

func (u *unlock) Release() {
	u.ch.Send()
}