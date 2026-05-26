#!/bin/bash

# Build script for simulation binaries

echo "=========================================="
echo "Building Simulation Binaries"
echo "=========================================="
echo ""

# Navigate to project root (two levels up from scripts)
cd "$(dirname "$0")/../.."

echo "Building bootstrap node..."
go build -o simulation/bin/bootstrap cmd/bootstrap/main.go
if [ $? -ne 0 ]; then
    echo "ERROR: Failed to build bootstrap node"
    exit 1
fi
echo "✓ Bootstrap node built"

echo "Building validator node..."
go build -o simulation/bin/node cmd/node/main.go
if [ $? -ne 0 ]; then
    echo "ERROR: Failed to build validator node"
    exit 1
fi
echo "✓ Validator node built"

echo "Building vote submission tool..."
go build -o simulation/bin/submit-vote cmd/submit-vote/main.go
if [ $? -ne 0 ]; then
    echo "ERROR: Failed to build vote submission tool"
    exit 1
fi
echo "✓ Vote submission tool built"

echo ""
echo "=========================================="
echo "Build Complete!"
echo "=========================================="
echo ""
echo "Binaries available in: simulation/bin/"
ls -lh simulation/bin/
