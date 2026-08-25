package main

import (
	"context"
	"log"
	"net"
)

func main() {
	pool := &ServerPool{}
	backendURLs := []string{
		"localhost:8081",
		"localhost:8082",
		"localhost:8083",
	}

	for _, url := range backendURLs {
		pool.AddBackend(&Backend{URL: url})
	}

	go HealthCheck(context.Background(), pool)

	listener, err := net.Listen("tcp", "localhost:8080")
	if err != nil {
		log.Fatalf("Cannot start listener: %v", err)
	}
	defer listener.Close()

	log.Printf("Load balancer listening on localhost:8080")
	for {
		connection, err := listener.Accept()
		if err != nil {
			log.Printf("Cannot accept connection: %v", err)
			continue
		}

		go HandleConnection(connection, pool)
	}
}
