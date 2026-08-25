package main

import "sync/atomic"

// ServerPool stores the backends used for round-robin load balancing.
type ServerPool struct {
	Backends []*Backend
	Current  uint64
}

// AddBackend adds a backend to the pool.
func (pool *ServerPool) AddBackend(backend *Backend) {
	pool.Backends = append(pool.Backends, backend)
}

// NextIndex advances and returns the next round-robin index.
func (pool *ServerPool) NextIndex() int {
	return int(atomic.AddUint64(&pool.Current, 1))
}

// GetNextPeer returns the next healthy backend, or nil when none are healthy.
func (pool *ServerPool) GetNextPeer() *Backend {
	backendCount := len(pool.Backends)
	if backendCount == 0 {
		return nil
	}

	startIndex := pool.NextIndex() % backendCount
	for offset := 0; offset < backendCount; offset++ {
		backend := pool.Backends[(startIndex+offset)%backendCount]
		if backend != nil && backend.IsAlive() {
			return backend
		}
	}

	return nil
}

// GetNextAvailablePeer returns and reserves the next healthy, unused backend.
func (pool *ServerPool) GetNextAvailablePeer() *Backend {
	backendCount := len(pool.Backends)
	if backendCount == 0 {
		return nil
	}

	startIndex := pool.NextIndex() % backendCount
	for offset := 0; offset < backendCount; offset++ {
		backend := pool.Backends[(startIndex+offset)%backendCount]
		if backend != nil && backend.TryAcquire() {
			return backend
		}
	}

	return nil
}
