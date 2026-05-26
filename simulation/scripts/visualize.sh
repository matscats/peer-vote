#!/bin/bash

# Script to generate network visualization from simulation logs

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "=========================================="
echo "Network Visualization Generator"
echo "=========================================="
echo ""

# Check if logs exist
if [ ! -d "../logs" ] || [ -z "$(ls -A ../logs 2>/dev/null)" ]; then
    echo "❌ No logs found in simulation/logs/"
    echo "   Please run a simulation first (e.g., ./quick_test.sh)"
    exit 1
fi

echo "📊 Analyzing logs..."
echo ""

# Build the visualization generator
echo "Building visualization generator..."
go build -o ../bin/visualize generate_visualization.go

if [ $? -ne 0 ]; then
    echo "❌ Failed to build visualization generator"
    exit 1
fi

echo "✅ Build successful"
echo ""

# Run the generator
echo "Generating visualization..."
../bin/visualize

if [ $? -ne 0 ]; then
    echo "❌ Failed to generate visualization"
    exit 1
fi

echo ""
echo "=========================================="
echo "✅ Visualization Complete!"
echo "=========================================="
echo ""
echo "Output files:"
echo "  📄 simulation/visualization.html"
echo "  📄 simulation/network_data.json"
echo ""
echo "To view the visualization:"
echo "  open ../visualization.html"
echo ""
echo "Or on Linux:"
echo "  xdg-open ../visualization.html"
echo ""
