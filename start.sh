#!/bin/bash

set -e

BACKEND_PID=""
CLEANED_UP=0

cleanup() {
	if [[ "$CLEANED_UP" -eq 1 ]]; then
		return
	fi
	CLEANED_UP=1

	if [[ -n "$BACKEND_PID" ]] && kill -0 "$BACKEND_PID" 2>/dev/null; then
		pkill -TERM -P "$BACKEND_PID" 2>/dev/null || true
		kill "$BACKEND_PID" 2>/dev/null || true
		wait "$BACKEND_PID" 2>/dev/null || true
	fi
}

trap cleanup SIGINT EXIT

go run test_backends/main.go &
BACKEND_PID=$!
go run .