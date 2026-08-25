package main

import (
	"context"
	"flag"
	"log"
	"net"
	"strconv"
	"strings"
)

func main() {
	port := flag.Int("port", 8080, "port for the load balancer to listen on")
	backendList := flag.String("backends", "http://localhost:8081,http://localhost:8082,http://localhost:8083", "comma-separated backend URLs")
	flag.Parse()

	pool := &ServerPool{}
	for _, url := range strings.Split(*backendList, ",") {
		url = strings.TrimSpace(url)
		url = strings.TrimPrefix(url, "http://")
		url = strings.TrimPrefix(url, "https://")
		if url != "" {
			pool.AddBackend(&Backend{URL: url})
		}
	}

	go HealthCheck(context.Background(), pool)

	listenAddress := "localhost:" + strconv.Itoa(*port)
	listener, err := net.Listen("tcp", listenAddress)
	if err != nil {
		log.Fatalf("Cannot start listener: %v", err)
	}
	defer listener.Close()

	log.Printf("Load balancer listening on %s", listenAddress)
	for {
		connection, err := listener.Accept()
		if err != nil {
			log.Printf("Cannot accept connection: %v", err)
			continue
		}

		go HandleConnection(connection, pool)
	}
}
