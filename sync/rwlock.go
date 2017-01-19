package chansync

import (
	"time"
)

// ReadWrite lock is a lock that can be used to control read and write access
// to a resource. A read lock cannot be acquired if the write lock is active. A
// read lock can be acquired if other read locks are active. A write lock
// cannot be acquired if any other locks are active.
//
// Simultaneous calls to AcquireRead or TryAcquireRead will block each other.
// This is to provide consistency and logical behavior.
//
// ReadUnlock and WriteUnlock objects must be discarded after any method is
// called. The only exception is ReadUnlock.TryPromote, if and only if it fails
// to promote the lock.
type ReadWriteLock interface {
	// AcquireRead blocks until a read lock can be acquired. AcquireRead
	// returns a ReadUnlock associated with the acquired lock.
	AcquireRead() ReadUnlock

	// TryAcquireRead attempts to acquire a read lock. If a lock cannot be
	// acquired, TryAcquireRead returns (nil, false). If a lock is acquired,
	// TryAcquireRead returns a (ReadUnlock, true) pair. The ReadUnlock is
	// associated with the acquired lock.
	TryAcquireRead() (ReadUnlock, bool)

	// AcquireWrite block until the write lock can be acquired. AcquireWrite
	// returns a WriteUnlock associated with the acquired lock.
	AcquireWrite() WriteUnlock

	// TryAcquireWrite attempts to acquire a write lock. If the lock cannot be
	// acquired, TryAcquireWrite returns (nil, false). If the lock is acquired,
	// TryAcquireWrite returns a (WriteUnlock, true) pair. The WriteUnlock is
	// associated with the acquired write.
	TryAcquireWrite() (WriteUnlock, bool)
}

// ReadUnlock represents a read lock acquired from an instance of
// ReadWriteLock. A read lock can be released, or promoted to a write lock.
type ReadUnlock interface {
	// Release releases the read lock.
	Release()

	// Promote block until the read lock can be promoted into a write lock.
	// Promote returns a WriteUnlock associated with the promoted lock.
	Promote() WriteUnlock

	// TryPromote attempts to promote the read lock into a write lock. Promote
	// returns a WriteUnlock associated with the promoted lock.
	TryPromote() (WriteUnlock, bool)
}

type WriteUnlock interface {
	// Release releases the write lock.
	Release()

	// Demote demotes the write lock into a read lock. Demote always returns
	// immediately.
	Demote() ReadUnlock
}


type rwlock struct {
	read, write Lock
	reads AtomicInt
	release SyncChannel
}

type runlock struct {
	lock *rwlock
}

type wunlock struct {
	unlock Unlock
	lock *rwlock
}

func NewReadWriteLock() ReadWriteLock {
	l := &rwlock{
		write: NewLock(),
		reads: NewAtomicInt(0),
		release: NewSyncChannel(),
	}

	return l
}

func (l *rwlock) AcquireRead() ReadUnlock {
	// see TryAcquireRead
	u1 := l.read.Acquire()
	defer u1.Release()

	// acquire the write lock
	u2 := l.write.Acquire()
	// release once the read count has been updated
	defer u2.Release()

	// increment the read count
	for !Increment(l.reads) {}

	// create the read unlock
	return l.newReadUnlock()
}

func (l *rwlock) TryAcquireRead() (ReadUnlock, bool) {
	// l.read is only locked in [Try]AcquireRead; acquiring l.read blocks so
	// that it's possible to differentiate between another [Try]AcquireRead
	// owning l.write and an active write lock owning l.write; if l.read was
	// not used or was not blocking, a TryAcquireRead call concurrent with
	// another [Try]AcquireRead call would have a chance of false failure; a
	// read lock acquisition should never block another read lock acquisition.
	u1 := l.read.Acquire()
	defer u1.Release()

	// try to acquire the write lock
	u2, ok := l.write.TryAcquire()
	if !ok {
		return nil, false
	}
	// release once the read count has been updated
	defer u2.Release()

	// increment the read count
	for !Increment(l.reads) {}

	// create the read unlock
	return l.newReadUnlock(), true
}

func (l *rwlock) AcquireWrite() WriteUnlock {
	// acquire the write lock
	u := l.write.Acquire()

	// wait until there are no active read locks
	for {
		if l.reads.Read() == 0 {
			break
		}

		// wait for a release notification, or a timeout; if this does not
		// have a timeout, there is a race condition with ReadUnlock.Release;
		// in rare situations, with the right timing, Recv could deadlock here
		l.release.TimeoutRecv(time.Millisecond)
	}

	// create the write unlock
	return l.newWriteUnlock(u)
}

func (l *rwlock) TryAcquireWrite() (WriteUnlock, bool) {
	// try to acquire the write lock
	u, ok := l.write.TryAcquire()
	if !ok {
		return nil, false
	}

	// check if there are any active read locks
	if l.reads.Read() != 0 {
		u.Release()
		return nil, false
	}

	// create the write unlock
	return l.newWriteUnlock(u), true
}

func (l *rwlock) newReadUnlock() ReadUnlock {
	return &runlock{
		lock: l,
	}
}

func (r *runlock) Promote() WriteUnlock {
	// acquire the write lock
	u := r.lock.write.Acquire()

	// wait until there are no other active read locks
	for {
		if r.lock.reads.Read() == 1 {
			break
		}

		// see AcquireWrite
		r.lock.release.TimeoutRecv(time.Millisecond)
	}

	// release this read lock
	if !r.lock.reads.Write(1, 0) {
		panic("ReadWriteLock has inconsistent internal state")
	}

	// create the write unlock
	return r.lock.newWriteUnlock(u)
}

func (r *runlock) TryPromote() (WriteUnlock, bool) {
	// try to acquire the write lock
	u, ok := r.lock.write.TryAcquire()
	if !ok {
		return nil, false
	}

	// check if there are any other active read locks
	if r.lock.reads.Read() != 1 {
		u.Release()
		return nil, false
	}

	// release this read lock
	if !r.lock.reads.Write(1, 0) {
		panic("ReadWriteLock has inconsistent internal state")
	}

	// create the write unlock
	return r.lock.newWriteUnlock(u), true
}

func (r *runlock) Release() {
	// release this read lock
	for !Decrement(r.lock.reads) {}

	// send a release event to a waiting write acquire
	r.lock.release.TrySend()
}

func (l *rwlock) newWriteUnlock(u Unlock) WriteUnlock {
	return &wunlock{
		unlock: u,
		lock: l,
	}
}

func (w *wunlock) Demote() ReadUnlock {
	// release this write lock once the read count has been updated
	defer w.unlock.Release()

	// increment the read count
	if !w.lock.reads.Write(0, 1) {
		panic("ReadWriteLock has inconsistent internal state")
	}

	// create the read unlock
	return w.lock.newReadUnlock()
}

func (w *wunlock) Release() {
	// release this write lock
	w.unlock.Release()
}