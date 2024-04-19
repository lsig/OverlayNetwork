#!/bin/bash

xterm -e "cd registry; go run registry.go" &

sleep 4

for i in {1..10}; do
  xterm -e "cd messages; go run messages.go localhost:8080" &
done
