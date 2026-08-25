package main

import "sync"

// Backend represents a downstream server and its current health status.
type Backend struct {
	URL   string
	Alive bool

	mu sync.RWMutex
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
