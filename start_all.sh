#!/bin/bash
export PATH=$PATH:/usr/local/go/bin
set -e
cd /home/lovewinner/codes/uptime_ng

echo "Starting backend..."
nohup ./server >> server.log 2>&1 &
BACK_PID=$!
echo "Backend PID: $BACK_PID"
sleep 2

if ! ps -p "$BACK_PID" > /dev/null 2>&1; then
  echo "Backend failed to start. Log:"
  tail -20 server.log
  exit 1
fi

echo "Starting frontend (vite)..."
cd web
nohup npm run dev -- --host >> ../vite.log 2>&1 &
VITE_PID=$!
echo "Vite PID: $VITE_PID"
sleep 2

if ! ps -p "$VITE_PID" > /dev/null 2>&1; then
  echo "Vite failed to start. Log:"
  tail -20 ../vite.log
  exit 1
fi

echo "All started. BACKEND=$BACK_PID VITE=$VITE_PID"
echo "Frontend: http://10.14.25.86:5173"
echo "Backend:  http://10.14.25.86:3000"
