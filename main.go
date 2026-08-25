package main

import (
	"context"
	"log"
	"net"
	"os"
	"strconv"

	"gopkg.in/yaml.v3"
)

type configuration struct {
	LoadBalancer struct {
		Port int `yaml:"port"`
	} `yaml:"load_balancer"`
	BackendSettings struct {
		DefaultTTL string `yaml:"default_ttl"`
	} `yaml:"backend_settings"`
	Backends []struct {
		Port int `yaml:"port"`
	} `yaml:"backends"`
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

	pool := &ServerPool{}
	for _, backend := range config.Backends {
		pool.AddBackend(&Backend{URL: "localhost:" + strconv.Itoa(backend.Port)})
	}

	go HealthCheck(context.Background(), pool)

	listenAddress := "localhost:" + strconv.Itoa(config.LoadBalancer.Port)
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
