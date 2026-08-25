#!/bin/bash

set -e

BACKEND_PORTS="${1:-8081,8082,8083}"
LB_PORT="${2:-8080}"
BACKEND_PID=""
CLEANED_UP=0

cleanup() {
	if [[ "$CLEANED_UP" -eq 1 ]]; then
		return
	fi
	CLEANED_UP=1

	if [[ -n "$BACKEND_PID" ]] && kill -0 "$BACKEND_PID" 2>/dev/null; then
		kill "$BACKEND_PID" 2>/dev/null || true
		wait "$BACKEND_PID" 2>/dev/null || true
	fi
}

trap cleanup SIGINT EXIT

(cd test_backends && go run main.go "-ports=$BACKEND_PORTS") &
BACKEND_PID=$!

FORMATTED_STRING=""
IFS=',' read -ra PORTS <<< "$BACKEND_PORTS"
for port in "${PORTS[@]}"; do
	port="$(printf '%s' "$port" | tr -d '[:space:]')"
	if [[ -n "$port" ]]; then
		if [[ -n "$FORMATTED_STRING" ]]; then
			FORMATTED_STRING+=","
		fi
		FORMATTED_STRING+="localhost:$port"
	fi
done

go run . "-port=$LB_PORT" "-backends=$FORMATTED_STRING"