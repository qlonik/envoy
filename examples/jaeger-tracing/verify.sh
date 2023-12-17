#!/bin/bash -e

export NAME=jaeger-tracing
export PORT_PROXY="${JAEGER_PORT_PROXY:-11010}"
export PORT_UI="${JAEGER_PORT_UI:-11011}"
export MANUAL=true

# shellcheck source=examples/verify-common.sh
. "$(dirname "${BASH_SOURCE[0]}")/../verify-common.sh"

run_log "Compile the go plugin library"
"${DOCKER_COMPOSE[@]}" -f docker-compose.compile.yaml up --quiet-pull --remove-orphans go_plugin_compile

run_log "Start all of our containers"
bring_up_example

wait_for 10 bash -c "responds_with Hello http://localhost:${PORT_PROXY}/trace/1"

run_log "Test services"
responds_with \
    Hello \
    "http://localhost:${PORT_PROXY}/trace/1"

run_log "Test Jaeger UI"
responds_with \
    "<!doctype html>" \
    "http://localhost:${PORT_UI}"
