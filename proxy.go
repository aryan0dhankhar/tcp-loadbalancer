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

	backend := pool.GetNextPeer()
	if backend == nil {
		log.Printf("Cannot proxy connection: no healthy backend available")
		return
	}

	backendConn, err := net.Dial("tcp", backend.URL)
	if err != nil {
		log.Printf("Cannot connect to backend %s: %v", backend.URL, err)
		return
	}
	defer backendConn.Close()

	var waitGroup sync.WaitGroup
	copyDone := make(chan struct{}, 2)
	waitGroup.Add(2)

	go func() {
		defer waitGroup.Done()
		_, _ = io.Copy(backendConn, clientConn)
		copyDone <- struct{}{}
	}()

	go func() {
		defer waitGroup.Done()
		_, _ = io.Copy(clientConn, backendConn)
		copyDone <- struct{}{}
	}()

	// Closing both connections unblocks the other copy when either direction ends.
	<-copyDone
	clientConn.Close()
	backendConn.Close()
	waitGroup.Wait()
}
