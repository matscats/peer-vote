#!/bin/bash

echo "=========================================="
echo "Starting Bootstrap Node"
echo "=========================================="
echo ""

# Build the bootstrap node
echo "Building bootstrap node..."
go build -o bin/bootstrap cmd/bootstrap/main.go
if [ $? -ne 0 ]; then
    echo "Build failed!"
    exit 1
fi

echo "Build successful!"
echo ""

# Start the bootstrap node
echo "Starting bootstrap node on port 4000..."
./bin/bootstrap -port 4000
