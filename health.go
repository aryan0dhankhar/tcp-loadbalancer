package main

import (
	"context"
	"log"
	"net"
	"time"
)

const (
	healthCheckInterval = 5 * time.Second
	healthCheckTimeout  = 3 * time.Second
)

// HealthCheck periodically probes every backend until the context is canceled.
func HealthCheck(ctx context.Context, pool *ServerPool) {
	if pool == nil {
		return
	}

	ticker := time.NewTicker(healthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for _, backend := range pool.Backends {
				if backend == nil {
					continue
				}

				connection, err := net.DialTimeout("tcp", backend.URL, healthCheckTimeout)
				if err != nil {
					if backend.IsAlive() {
						log.Printf("Backend %s is down: %v", backend.URL, err)
					}
					backend.SetAlive(false)
					continue
				}

				connection.Close()
				if !backend.IsAlive() {
					log.Printf("Backend %s is up", backend.URL)
				}
				backend.SetAlive(true)
			}
		}
	}
}
