package main

import (
	"net"
	"testing"
)

func TestServerPoolDoesNotAssignBackendTwice(t *testing.T) {
	first := &Backend{URL: "localhost:8081"}
	first.SetAlive(true)
	second := &Backend{URL: "localhost:8082"}
	second.SetAlive(true)

	pool := &ServerPool{}
	pool.AddBackend(first)
	pool.AddBackend(second)

	leased := pool.GetNextAvailablePeer()
	if leased == nil {
		t.Fatal("expected the first backend lease")
	}
	defer leased.Release()

	other := pool.GetNextAvailablePeer()
	if other == nil {
		t.Fatal("expected the second backend lease")
	}
	defer other.Release()
	if other == leased {
		t.Fatal("assigned the same backend to two active processes")
	}

	if pool.GetNextAvailablePeer() != nil {
		t.Fatal("expected no backend while all ports are leased")
	}
}

func TestServerPoolFailsOverAfterBackendPortStops(t *testing.T) {
	firstListener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}
	firstAddress := firstListener.Addr().String()
	firstListener.Close()

	secondListener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}
	defer secondListener.Close()

	first := &Backend{URL: firstAddress}
	first.SetAlive(true)
	second := &Backend{URL: secondListener.Addr().String()}
	second.SetAlive(true)
	pool := &ServerPool{}
	pool.AddBackend(first)
	pool.AddBackend(second)

	first.SetAlive(false)
	fallback := pool.GetNextAvailablePeer()
	if fallback != second {
		t.Fatalf("expected failover to %s, got %v", second.URL, fallback)
	}
	fallback.Release()

	replacementListener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}
	defer replacementListener.Close()
	replacement := &Backend{URL: replacementListener.Addr().String()}
	replacement.SetAlive(true)
	pool.AddBackend(replacement)

	second.SetAlive(false)
	leased := pool.GetNextAvailablePeer()
	if leased != replacement {
		t.Fatalf("expected reassignment to replacement port %s, got %v", replacement.URL, leased)
	}
	leased.Release()
}
