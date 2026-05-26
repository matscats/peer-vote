#!/bin/bash

# Script to generate voter keypairs for election simulation

VOTERS_DIR="../keys/voters"
NUM_VOTERS=${1:-10}

echo "=========================================="
echo "Generating Voter Keypairs"
echo "=========================================="
echo ""

# Create voters directory
mkdir -p "$VOTERS_DIR"

# Navigate to project root
cd "$(dirname "$0")/../.."

echo "Generating $NUM_VOTERS voter keypairs..."
echo ""

for i in $(seq 1 $NUM_VOTERS); do
    VOTER_ID="voter$(printf "%03d" $i)"
    KEY_PATH="simulation/keys/voters/${VOTER_ID}.key"
    
    # Generate keypair
    ./simulation/bin/node -generate-key -key-path "$KEY_PATH" > /dev/null 2>&1
    
    if [ $? -eq 0 ]; then
        # Extract public key
        PUBKEY=$(./simulation/bin/node -generate-key -key-path "${KEY_PATH}.tmp" 2>&1 | grep "Public key:" | awk '{print $3}')
        rm -f "${KEY_PATH}.tmp"
        
        echo "✓ Generated $VOTER_ID (key: $KEY_PATH)"
    else
        echo "✗ Failed to generate $VOTER_ID"
    fi
done

echo ""
echo "=========================================="
echo "Voter Generation Complete!"
echo "=========================================="
echo ""
echo "Generated $NUM_VOTERS voter keypairs in: $VOTERS_DIR"
echo ""
echo "To add voters to eligibility list, run:"
echo "  ./update_eligibility.sh"
