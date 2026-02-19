#!/bin/bash
echo "Finding and stopping processes on port 8080..."

# Find PIDs using port 8080
PIDS=$(lsof -t -i:8080 2>/dev/null || ss -tlnp | grep :8080 | sed -n 's/.*pid=\([0-9]*\).*/\1/p')

if [ -z "$PIDS" ]; then
    echo "No process found on port 8080"
    exit 0
fi

for PID in $PIDS; do
    echo "Killing process $PID"
    kill -9 $PID 2>/dev/null
done

echo "Port 8080 is now free"
