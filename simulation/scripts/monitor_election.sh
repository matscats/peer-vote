#!/bin/bash

# Real-time election monitoring script

LOGS_DIR="../logs"

echo "=========================================="
echo "Real-Time Election Monitor"
echo "=========================================="
echo ""
echo "Press Ctrl+C to exit"
echo ""

# Function to get latest stats
show_stats() {
    clear
    echo "=========================================="
    echo "Election Monitor - $(date '+%H:%M:%S')"
    echo "=========================================="
    echo ""
    
    # Network status
    echo "🌐 Network Status:"
    BOOTSTRAP_RUNNING=$(ps aux | grep "bin/bootstrap" | grep -v grep | wc -l)
    NODES_RUNNING=$(ps aux | grep "bin/node" | grep -v grep | wc -l)
    echo "  Bootstrap: $([ $BOOTSTRAP_RUNNING -gt 0 ] && echo '✅ Running' || echo '❌ Stopped')"
    echo "  Validator nodes: $NODES_RUNNING/3"
    
    if [ -f "$LOGS_DIR/node1.log" ]; then
        PEERS=$(grep -h "Connected peers:" $LOGS_DIR/node*.log 2>/dev/null | tail -1 | awk '{print $3}')
        echo "  Connected peers: ${PEERS:-0}"
    fi
    echo ""
    
    # Blockchain status
    echo "📦 Blockchain Status:"
    if [ -f "$LOGS_DIR/node1.log" ]; then
        LATEST_HEIGHT=$(grep -h "finalized successfully" $LOGS_DIR/node*.log 2>/dev/null | tail -1 | grep -o "height [0-9]*" | awk '{print $2}')
        TOTAL_BLOCKS=$(grep -h "finalized successfully" $LOGS_DIR/node*.log 2>/dev/null | wc -l)
        echo "  Latest block: #${LATEST_HEIGHT:-0}"
        echo "  Total finalizations: $TOTAL_BLOCKS"
        
        # Calculate votes in blocks
        VOTES_FINALIZED=$(grep -h "finalized successfully" $LOGS_DIR/node*.log 2>/dev/null | grep -o "voted count: [0-9]*" | awk '{sum += $3} END {print sum}')
        echo "  Votes finalized: ${VOTES_FINALIZED:-0}"
    else
        echo "  (No data yet)"
    fi
    echo ""
    
    # Vote submissions
    echo "🗳️  Vote Submissions:"
    VOTE_LOGS=$(ls -1 $LOGS_DIR/vote_*.log 2>/dev/null | wc -l)
    echo "  Total submitted: $VOTE_LOGS"
    
    if [ $VOTE_LOGS -gt 0 ]; then
        echo ""
        echo "  By candidate:"
        for candidate in candidate-a candidate-b candidate-c; do
            COUNT=$(grep -l "choice=$candidate" $LOGS_DIR/vote_*.log 2>/dev/null | wc -l)
            PERCENTAGE=$(awk "BEGIN {printf \"%.1f\", ($COUNT / $VOTE_LOGS) * 100}")
            BAR=$(printf '█%.0s' $(seq 1 $(($COUNT / 2))))
            echo "    $candidate: $COUNT ($PERCENTAGE%) $BAR"
        done
    fi
    echo ""
    
    # Recent activity
    echo "📝 Recent Activity:"
    if [ -f "$LOGS_DIR/node1.log" ]; then
        echo "  Last 3 finalized blocks:"
        grep -h "finalized successfully" $LOGS_DIR/node*.log 2>/dev/null | tail -3 | while read line; do
            HEIGHT=$(echo "$line" | grep -o "height [0-9]*" | awk '{print $2}')
            VOTES=$(echo "$line" | grep -o "voted count: [0-9]*" | awk '{print $3}')
            TIME=$(echo "$line" | awk '{print $1, $2}')
            echo "    [$TIME] Block #$HEIGHT with $VOTES votes"
        done
    fi
    echo ""
    
    # Errors
    ERRORS=$(grep -h "ERROR" $LOGS_DIR/node*.log 2>/dev/null | grep -v "no active round" | wc -l)
    if [ $ERRORS -gt 0 ]; then
        echo "⚠️  Errors: $ERRORS (check logs for details)"
    else
        echo "✅ No critical errors"
    fi
    
    echo ""
    echo "────────────────────────────────────────"
    echo "Refreshing in 2 seconds..."
}

# Monitor loop
while true; do
    show_stats
    sleep 2
done
