package main

import (
	"bufio"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"gopkg.in/yaml.v3"
)

type configuration struct {
	BackendSettings struct {
		DefaultTTL string `yaml:"default_ttl"`
		MaxTTL     string `yaml:"max_ttl"`
	} `yaml:"backend_settings"`
	Backends []struct {
		Port int `yaml:"port"`
	} `yaml:"backends"`
}

var nextProcessID uint64

type processRegistry struct {
	mu          sync.RWMutex
	connections map[uint64]net.Conn
}

var activeProcesses = processRegistry{connections: make(map[uint64]net.Conn)}

func (registry *processRegistry) add(id uint64, connection net.Conn) {
	registry.mu.Lock()
	registry.connections[id] = connection
	registry.mu.Unlock()
}

func (registry *processRegistry) remove(id uint64) {
	registry.mu.Lock()
	delete(registry.connections, id)
	registry.mu.Unlock()
}

// CloseProcess forcefully closes an active process connection by its ID.
func CloseProcess(id uint64) bool {
	activeProcesses.mu.RLock()
	connection, exists := activeProcesses.connections[id]
	activeProcesses.mu.RUnlock()
	if !exists {
		return false
	}

	return connection.Close() == nil
}

func handleProcess(connection net.Conn, port, defaultTTL, maxTTL string) {
	defer connection.Close()
	processID := atomic.AddUint64(&nextProcessID, 1)
	activeProcesses.add(processID, connection)
	defer activeProcesses.remove(processID)

	reader := bufio.NewReader(connection)
	_ = connection.SetReadDeadline(time.Now().Add(time.Second))
	request, _ := reader.ReadString('\n')
	_ = connection.SetReadDeadline(time.Time{})

	requestLine := strings.TrimSpace(request)
	if strings.HasPrefix(requestLine, "HEALTH") {
		_, _ = connection.Write([]byte("HEALTHY\n"))
		return
	}

	requestedTTL := strings.TrimSpace(strings.TrimPrefix(requestLine, "TTL="))
	if requestedTTL == "" {
		requestedTTL = defaultTTL
	}

	calculatedTTL, err := time.ParseDuration(requestedTTL)
	if err != nil {
		log.Printf("Invalid TTL %q for backend %s; using default %s", requestedTTL, port, defaultTTL)
		calculatedTTL, err = time.ParseDuration(defaultTTL)
		if err != nil {
			log.Printf("Invalid default TTL %q; closing process on port %s", defaultTTL, port)
			return
		}
	}

	maximumTTL, err := time.ParseDuration(maxTTL)
	if err == nil && calculatedTTL > maximumTTL {
		calculatedTTL = maximumTTL
	}

	log.Printf("Process %d assigned to port %s with TTL %s", processID, port, calculatedTTL)
	_, _ = connection.Write([]byte("Process " + strconv.FormatUint(processID, 10) + " assigned to port " + port + ". TTL set to " + calculatedTTL.String() + "\n"))
	ttlTimer := time.NewTimer(calculatedTTL)
	defer ttlTimer.Stop()
	<-ttlTimer.C
	_, _ = connection.Write([]byte("TTL expired. Kernel freeing process resources...\n"))
	log.Printf("Process %d on port %s expired", processID, port)
}

func runBackend(address, port, defaultTTL, maxTTL string, waitGroup *sync.WaitGroup) {
	defer waitGroup.Done()

	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("Cannot start backend on %s: %v", address, err)
	}

	log.Printf("Backend listening on %s", address)
	for {
		connection, err := listener.Accept()
		if err != nil {
			log.Printf("Cannot accept connection on %s: %v", address, err)
			continue
		}

		go handleProcess(connection, port, defaultTTL, maxTTL)
	}
}

func main() {
	configFile, err := os.ReadFile("ports.yaml")
	if err != nil {
		log.Fatalf("Cannot read ports.yaml: %v", err)
	}

	var config configuration
	if err := yaml.Unmarshal(configFile, &config); err != nil {
		log.Fatalf("Cannot parse ports.yaml: %v", err)
	}

	var waitGroup sync.WaitGroup
	for _, backend := range config.Backends {
		port := strconv.Itoa(backend.Port)
		waitGroup.Add(1)
		go runBackend("localhost:"+port, port, config.BackendSettings.DefaultTTL, config.BackendSettings.MaxTTL, &waitGroup)
	}

	waitGroup.Wait()
}
