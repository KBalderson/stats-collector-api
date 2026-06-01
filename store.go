package main

import (
	"sync"
	"time"
)

type Stats struct {
	DeviceId   string
	SentAt     time.Time
	UploadTime int
}

type Heartbeat struct { // In typescript, I would allow this to be a partial, but i don't see that option in go, so we will just have a different struct for the request body
	DeviceId string
	SentAt   time.Time
}

// Store holds the in-memory heartbeats and stats. Every HTTP request is served
// on its own goroutine, so all access goes through the RWMutex: writers take the
// exclusive Lock, readers take the shared RLock (many readers, no writer).
type Store struct {
	heartbeatsMutex sync.RWMutex
	heartbeats      []Heartbeat
	statsMutex      sync.RWMutex
	stats           []Stats
}

// *Pointer
// &Reference
func NewStore() *Store {
	return &Store{
		heartbeats: make([]Heartbeat, 0),
		stats:      make([]Stats, 0),
	}
}

func (s *Store) AddHeartbeat(h Heartbeat) {
	s.heartbeatsMutex.Lock()
	defer s.heartbeatsMutex.Unlock()
	s.heartbeats = append(s.heartbeats, h)
}

func (s *Store) AddStats(st Stats) {
	s.statsMutex.Lock()
	defer s.statsMutex.Unlock()
	s.stats = append(s.stats, st)
}

// HeartbeatsFor returns a snapshot copy of one device's heartbeats. Filtering
// happens under the read lock, and the returned slice is freshly allocated, so
// the caller can use it after the lock is released without racing a writer.
func (s *Store) HeartbeatsFor(deviceId string) []Heartbeat {
	s.heartbeatsMutex.RLock()
	defer s.heartbeatsMutex.RUnlock()
	heartbeats := make([]Heartbeat, 0)
	for _, h := range s.heartbeats {
		if h.DeviceId == deviceId {
			heartbeats = append(heartbeats, h)
		}
	}
	return heartbeats
}

// StatsFor returns a snapshot copy of one device's stats.
func (s *Store) StatsFor(deviceId string) []Stats {
	s.statsMutex.RLock()
	defer s.statsMutex.RUnlock()
	stats := make([]Stats, 0)
	for _, st := range s.stats {
		if st.DeviceId == deviceId {
			stats = append(stats, st)
		}
	}
	return stats
}
