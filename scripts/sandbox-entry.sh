#!/bin/bash
set -e

BLUEPRINT=""
BLUEPRINT_FILE=""
TASK=""
ADAPTER="${FORGE_ADAPTER:-echo}"
HARNESS_PORT="${FORGE_HARNESS_PORT:-50051}"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --blueprint) BLUEPRINT="$2"; shift 2 ;;
    --blueprint-file) BLUEPRINT_FILE="$2"; shift 2 ;;
    --task) TASK="$2"; shift 2 ;;
    --adapter) ADAPTER="$2"; shift 2 ;;
    *) echo "Unknown arg: $1"; exit 1 ;;
  esac
done

cd /opt/forge/harness
FORGE_ADAPTER="$ADAPTER" FORGE_HARNESS_PORT="$HARNESS_PORT" node dist/server.js &
HARNESS_PID=$!

for i in $(seq 1 20); do
  if command -v nc >/dev/null 2>&1 && nc -z localhost "$HARNESS_PORT" 2>/dev/null; then
    break
  fi
  sleep 0.5
done

cd /workspace

RUN_ARGS="blueprint run"
if [ -n "$BLUEPRINT_FILE" ]; then
  RUN_ARGS="$RUN_ARGS $BLUEPRINT_FILE"
elif [ -n "$BLUEPRINT" ]; then
  RUN_ARGS="$RUN_ARGS --builtin $BLUEPRINT"
fi
RUN_ARGS="$RUN_ARGS --harness localhost:$HARNESS_PORT"

forge $RUN_ARGS
EXIT_CODE=$?

kill "$HARNESS_PID" 2>/dev/null || true
exit $EXIT_CODE
