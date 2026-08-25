package main

import (
	"log"
	"net"
	"sync"
)

func runBackend(address, message string, waitGroup *sync.WaitGroup) {
	defer waitGroup.Done()

	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("Cannot start backend on %s: %v", address, err)
	}
	defer listener.Close()

	log.Printf("Backend listening on %s", address)
	for {
		connection, err := listener.Accept()
		if err != nil {
			log.Printf("Cannot accept connection on %s: %v", address, err)
			continue
		}

		go func(connection net.Conn) {
			defer connection.Close()
			if _, err := connection.Write([]byte(message)); err != nil {
				log.Printf("Cannot write response from %s: %v", address, err)
			}
		}(connection)
	}
}

func main() {
	var waitGroup sync.WaitGroup
	waitGroup.Add(3)

	go runBackend("localhost:8081", "Hello from backend 8081\n", &waitGroup)
	go runBackend("localhost:8082", "Hello from backend 8082\n", &waitGroup)
	go runBackend("localhost:8083", "Hello from backend 8083\n", &waitGroup)

	waitGroup.Wait()
}
