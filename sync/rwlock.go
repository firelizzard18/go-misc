package chansync

import (
	"time"
)


type ReadWriteLock interface {
	AcquireRead() ReadUnlock
	TryAcquireRead() (ReadUnlock, bool)
	AcquireWrite() WriteUnlock
	TryAcquireWrite() (WriteUnlock, bool)
}

type ReadUnlock interface {
	Release()
	Promote() WriteUnlock
	TryPromote() (WriteUnlock, bool)
}

type WriteUnlock interface {
	Release()
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