package main

import (
	"bufio"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
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

func handleProcess(connection net.Conn, port, defaultTTL, maxTTL string) {
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

	_, _ = connection.Write([]byte("Process assigned to port " + port + ". TTL set to " + calculatedTTL.String() + "\n"))
	ttlTimer := time.NewTimer(calculatedTTL)
	defer ttlTimer.Stop()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ttlTimer.C:
			_, _ = connection.Write([]byte("TTL expired. Kernel freeing process resources...\n"))
			return
		case <-ticker.C:
			if _, err := connection.Write([]byte("Process running...\n")); err != nil {
				return
			}
		}
	}
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
	configFile, err := os.ReadFile("config.yaml")
	if err != nil {
		log.Fatalf("Cannot read config.yaml: %v", err)
	}

	var config configuration
	if err := yaml.Unmarshal(configFile, &config); err != nil {
		log.Fatalf("Cannot parse config.yaml: %v", err)
	}

	var waitGroup sync.WaitGroup
	for _, backend := range config.Backends {
		port := strconv.Itoa(backend.Port)
		waitGroup.Add(1)
		go runBackend("localhost:"+port, port, config.BackendSettings.DefaultTTL, config.BackendSettings.MaxTTL, &waitGroup)
	}

	waitGroup.Wait()
}
