#!/bin/bash

# Run registry.go in a separate terminal
xterm -e "cd registry; go run registry.go" &

# Run message.go ten times in separate terminals
for i in {1..10}; do
  xterm -e "cd messages; go run messages.go localhost:8080" &
done

# Wait for background processes to finish (optional)
# wait
