#!/bin/bash -e

export NAME=opentelemetry
export PORT_PROXY="${OPENTELEMETRY_PORT_PROXY:-12000}"
export PORT_UI="${OPENTELEMETRY_PORT_UI:-12001}"
export MANUAL=true

# shellcheck source=examples/verify-common.sh
. "$(dirname "${BASH_SOURCE[0]}")/../verify-common.sh"

run_log "Compile the go plugin library"
"${DOCKER_COMPOSE[@]}" -f docker-compose.compile.yaml up --quiet-pull --remove-orphans go_plugin_compile

run_log "Start all of our containers"
bring_up_example

sleep 2

run_log "Make a request to service-1"
responds_with \
    "Hello from behind Envoy (service 1)!" \
    "http://localhost:${PORT_PROXY}/trace/1"

run_log "Make a request to service-2"
responds_with \
    "Hello from behind Envoy (service 2)!" \
    "http://localhost:${PORT_PROXY}/trace/2"

run_log "View the traces in OpenTelemetry UI"
responds_with \
    "<!DOCTYPE html>" \
    "http://localhost:${PORT_UI}/debug/tracez"

run_log "Service logs"
"${DOCKER_COMPOSE[@]}" logs envoy-front-proxy opentelemetry
