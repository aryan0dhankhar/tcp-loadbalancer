package main

import (
	"bufio"
	"log"
	"net"
	"os"
	"sort"
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
	} `yaml:"backend_settings"`
	Backends []struct {
		Port int `yaml:"port"`
	} `yaml:"backends"`
}

var nextProcessID uint64

type processRegistry struct {
	mu          sync.RWMutex
	connections map[uint64]process
}

type process struct {
	id         uint64
	port       string
	connection net.Conn
}

var activeProcesses = processRegistry{connections: make(map[uint64]process)}

func (registry *processRegistry) add(process process) {
	registry.mu.Lock()
	registry.connections[process.id] = process
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
	activeProcess, exists := activeProcesses.connections[id]
	activeProcesses.mu.RUnlock()
	if !exists {
		return false
	}

	return activeProcess.connection.Close() == nil
}

func statusMessage() string {
	activeProcesses.mu.RLock()
	processes := make([]process, 0, len(activeProcesses.connections))
	for _, activeProcess := range activeProcesses.connections {
		processes = append(processes, activeProcess)
	}
	activeProcesses.mu.RUnlock()

	sort.Slice(processes, func(first, second int) bool {
		firstPort, _ := strconv.Atoi(processes[first].port)
		secondPort, _ := strconv.Atoi(processes[second].port)
		return firstPort < secondPort
	})

	var builder strings.Builder
	for _, activeProcess := range processes {
		builder.WriteString("Process ")
		builder.WriteString(strconv.FormatUint(activeProcess.id, 10))
		builder.WriteString(" on port ")
		builder.WriteString(activeProcess.port)
		builder.WriteByte('\n')
	}
	if builder.Len() == 0 {
		return "No active processes\n"
	}
	return builder.String()
}

func handleProcess(connection net.Conn, port, defaultTTL string) {
	defer connection.Close()

	reader := bufio.NewReader(connection)
	_ = connection.SetReadDeadline(time.Now().Add(time.Second))
	request, _ := reader.ReadString('\n')
	_ = connection.SetReadDeadline(time.Time{})

	requestLine := strings.TrimSpace(request)
	if strings.HasPrefix(requestLine, "HEALTH") {
		_, _ = connection.Write([]byte("HEALTHY\n"))
		return
	}
	if requestLine == "STATUS" {
		_, _ = connection.Write([]byte(statusMessage()))
		return
	}

	calculatedTTL, err := time.ParseDuration(defaultTTL)
	if err != nil {
		log.Printf("Invalid default TTL %q; closing process on port %s", defaultTTL, port)
		return
	}

	processID := atomic.AddUint64(&nextProcessID, 1)
	activeProcesses.add(process{id: processID, port: port, connection: connection})
	defer activeProcesses.remove(processID)

	log.Printf("Process %d assigned to port %s with TTL %s", processID, port, calculatedTTL)
	_, _ = connection.Write([]byte("Process " + strconv.FormatUint(processID, 10) + " assigned to port " + port + ". TTL set to " + calculatedTTL.String() + "\n"))
	ttlTimer := time.NewTimer(calculatedTTL)
	defer ttlTimer.Stop()
	<-ttlTimer.C
	_, _ = connection.Write([]byte("TTL expired. Kernel freeing process resources...\n"))
	log.Printf("Process %d on port %s expired", processID, port)
}

func runBackend(address, port, defaultTTL string, waitGroup *sync.WaitGroup) {
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

		go handleProcess(connection, port, defaultTTL)
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
		go runBackend("localhost:"+port, port, config.BackendSettings.DefaultTTL, &waitGroup)
	}

	waitGroup.Wait()
}
