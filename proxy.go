package main

import (
	"io"
	"log"
	"net"
	"sync"
)

// HandleConnection proxies traffic between a client and a healthy backend.
func HandleConnection(clientConn net.Conn, pool *ServerPool) {
	if clientConn == nil {
		return
	}
	defer clientConn.Close()

	if pool == nil {
		log.Printf("Cannot proxy connection: server pool is nil")
		return
	}

	backend := pool.GetNextAvailablePeer()
	if backend == nil {
		log.Printf("Cannot proxy connection: no available backend port")
		return
	}
	defer backend.Release()

	backendConn, err := net.Dial("tcp", backend.URL)
	if err != nil {
		log.Printf("Cannot connect to backend %s: %v", backend.URL, err)
		return
	}
	defer backendConn.Close()

	var waitGroup sync.WaitGroup
	copyDone := make(chan string, 2)
	waitGroup.Add(2)

	go func() {
		defer waitGroup.Done()
		_, _ = io.Copy(backendConn, clientConn)
		copyDone <- "client"
	}()

	go func() {
		defer waitGroup.Done()
		_, _ = io.Copy(clientConn, backendConn)
		copyDone <- "backend"
	}()

	// Preserve the other direction after an input EOF so request/response traffic
	// can still return its response before both connections are closed.
	if completed := <-copyDone; completed == "client" {
		closeWrite(backendConn)
	} else {
		closeWrite(clientConn)
		backendConn.Close()
	}
	waitGroup.Wait()
}

func closeWrite(connection net.Conn) {
	if tcpConnection, ok := connection.(*net.TCPConn); ok {
		_ = tcpConnection.CloseWrite()
	}
}
