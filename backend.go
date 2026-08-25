package main

import "sync"

// Backend represents a downstream server and its current health status.
type Backend struct {
	URL   string
	Alive bool

	mu    sync.RWMutex
	inUse bool
}

// SetAlive updates the backend health status safely for concurrent callers.
func (backend *Backend) SetAlive(alive bool) {
	backend.mu.Lock()
	backend.Alive = alive
	backend.mu.Unlock()
}

// IsAlive reads the backend health status safely for concurrent callers.
func (backend *Backend) IsAlive() bool {
	backend.mu.RLock()
	alive := backend.Alive
	backend.mu.RUnlock()
	return alive
}

// TryAcquire reserves this backend for one active client connection.
func (backend *Backend) TryAcquire() bool {
	backend.mu.Lock()
	defer backend.mu.Unlock()

	if backend.inUse || !backend.Alive {
		return false
	}

	backend.inUse = true
	return true
}

// Release makes this backend available for another client connection.
func (backend *Backend) Release() {
	backend.mu.Lock()
	backend.inUse = false
	backend.mu.Unlock()
}
