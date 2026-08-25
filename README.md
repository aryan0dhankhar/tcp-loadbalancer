# TCP Load Balancer

A lightweight Layer-4 TCP load balancer written with Go's standard networking APIs and `gopkg.in/yaml.v3` for configuration.

## Features

- Round-robin selection of healthy backend ports
- One active leased process per backend port
- Automatic backend health checks
- Fixed TTL for every leased process
- Process IDs and active-process status queries
- Backend failover when a port becomes unhealthy

## Requirements

- Go 1.27 or newer
- `nc` (netcat) for manual testing

## Configuration

Edit [ports.yaml](ports.yaml):

```yaml
load_balancer:
  port: 8080
backend_settings:
  default_ttl: "90s"
backends:
  - port: 9001
  - port: 9002
  - port: 9003
  - port: 9004
  - port: 9005
```

`default_ttl` is applied to every client process. Backend ports do not have individual TTLs and remain open for future connections.

## Run

From the repository root:

```bash
./start.sh
```

The script starts the test backends in the background and the load balancer in the foreground. Press `Ctrl+C` to stop both.

You can also run them separately:

```bash
go run test_backends/main.go
```

In another terminal:

```bash
go run .
```

Allow the first health check to complete before creating client connections.

## Test A Connection

Connect to the load balancer:

```bash
nc localhost 8080
```

Type any line and press Enter. The connection receives a process ID and backend port, then remains active until the configured TTL expires.

Keep the client connection open. Closing `nc` releases the backend lease immediately.

## Test Concurrent Leases

The following command opens five connections concurrently, matching the five configured backend ports:

```bash
for i in 1 2 3 4 5; do
  (printf '\n'; tail -f /dev/null) | nc -w 100 localhost 8080 &
done
```

Each connection should receive a different backend port. A sixth simultaneous connection is rejected because all backend ports are leased.

## Query Active Processes

Query the process registry through any configured backend port:

```bash
printf 'STATUS\n' | nc -w 3 localhost 9001
```

The response lists active process IDs sorted by backend port:

```text
Process 1 on port 9001
Process 4 on port 9002
Process 2 on port 9005
```

## Run Failover Tests

Run the pool leasing and failover tests:

```bash
go test -run 'TestServerPool' ./...
```

Run all tests with the race detector:

```bash
go test ./... -race
```

The tests verify that a backend port cannot be leased twice and that an unhealthy backend is skipped in favor of another available backend.

## Protocol Notes

The load balancer uses raw TCP. `nc` command-line flags configure netcat itself; they are not sent to the server. Client input is only used to initiate a process, while the process TTL always comes from `ports.yaml`.
