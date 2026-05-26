#!/bin/bash

# Script to update eligibility.json with all generated voters

VOTERS_DIR="../keys/voters"
ELIGIBILITY_FILE="../configs/eligibility.json"

echo "=========================================="
echo "Updating Eligibility List"
echo "=========================================="
echo ""

cd "$(dirname "$0")"

# Check if voters directory exists
if [ ! -d "$VOTERS_DIR" ]; then
    echo "ERROR: Voters directory not found: $VOTERS_DIR"
    echo "Run ./generate_voters.sh first"
    exit 1
fi

# Count voter keys
VOTER_COUNT=$(ls -1 "$VOTERS_DIR"/*.key 2>/dev/null | wc -l)

if [ $VOTER_COUNT -eq 0 ]; then
    echo "ERROR: No voter keys found in $VOTERS_DIR"
    echo "Run ./generate_voters.sh first"
    exit 1
fi

echo "Found $VOTER_COUNT voter keys"
echo "Creating eligibility list..."

# Start JSON
echo '{' > "$ELIGIBILITY_FILE"
echo '  "eligible_voters": [' >> "$ELIGIBILITY_FILE"

# Add each voter
FIRST=true
for key_file in "$VOTERS_DIR"/*.key; do
    VOTER_ID=$(basename "$key_file" .key)
    
    if [ "$FIRST" = true ]; then
        echo "    \"$VOTER_ID\"" >> "$ELIGIBILITY_FILE"
        FIRST=false
    else
        echo "    ,\"$VOTER_ID\"" >> "$ELIGIBILITY_FILE"
    fi
done

# Close JSON
echo '  ]' >> "$ELIGIBILITY_FILE"
echo '}' >> "$ELIGIBILITY_FILE"

echo ""
echo "✓ Eligibility list updated: $ELIGIBILITY_FILE"
echo "✓ Added $VOTER_COUNT eligible voters"
echo ""
echo "Eligible voters:"
cat "$ELIGIBILITY_FILE"
