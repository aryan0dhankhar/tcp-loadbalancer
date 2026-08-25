package main

import (
	"flag"
	"log"
	"net"
	"strings"
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
	ports := flag.String("ports", "8081,8082,8083", "comma-separated backend ports")
	flag.Parse()

	backendPorts := strings.Split(*ports, ",")
	var waitGroup sync.WaitGroup
	for _, port := range backendPorts {
		port = strings.TrimSpace(port)
		if port == "" {
			continue
		}

		waitGroup.Add(1)
		go runBackend("localhost:"+port, "Hello from backend "+port+"\n", &waitGroup)
	}

	waitGroup.Wait()
}
