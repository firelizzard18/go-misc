package chansync

// Sync is a synchronization tool that can be used to synchronize execution of
// two concurrent routines.
type Sync interface {
	// SyncLeft returns immediately if a call to SyncRight is blocked.
	// Otherwise, SyncLeft blocks until SyncRight is called.
	SyncLeft()
	// TrySyncLeft returns true if a call to SyncRight is blocked. Otherwise
	// TrySyncLeft returns false.
	TrySyncLeft() bool
	// SyncRight returns immediately if a call to SyncLeft is blocked.
	// Otherwise, SyncRight blocks until SyncLeft is called.
	SyncRight()
	// TrySyncRight returns true if a call to SyncLeft is blocked. Otherwise
	// TrySyncRight returns false.
	TrySyncRight() bool
}


type syncOnce struct {
	ch SyncChannel
}

// NewSync returns a new Sync.
func NewSync() Sync {
	return &syncOnce{
		ch: NewSyncChannel(),
	}
}

func (s *syncOnce) SyncLeft() {
	s.ch.Send()
}

func (s *syncOnce) TrySyncLeft() bool {
	return s.ch.TrySend() == ChannelOpSuccess
}

func (s *syncOnce) SyncRight() {
	s.ch.Recv()
}

func (s *syncOnce) TrySyncRight() bool {
	return s.ch.TryRecv() == ChannelOpSuccess
}