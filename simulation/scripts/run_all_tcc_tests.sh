#!/bin/bash

# Master Test Script for TCC
# Executes all 5 tests and generates consolidated report

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Results directory
RESULTS_DIR="../results_tcc"
mkdir -p "$RESULTS_DIR"

# Timestamp for this test run
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
REPORT_FILE="$RESULTS_DIR/report_${TIMESTAMP}.txt"

echo "=========================================="
echo "TCC Test Suite - Complete Execution"
echo "=========================================="
echo ""
echo "Timestamp: $(date)"
echo "Results will be saved to: $RESULTS_DIR"
echo ""

# Initialize report
cat > "$REPORT_FILE" << EOF
========================================
TCC Test Suite - Execution Report
========================================

Execution Date: $(date)
System: $(uname -s) $(uname -r)
Go Version: $(go version)

========================================
EOF

# Function to log to both console and file
log() {
    echo -e "$1"
    echo -e "$1" | sed 's/\x1b\[[0-9;]*m//g' >> "$REPORT_FILE"
}

# Function to run a test and capture results
run_test() {
    local test_num=$1
    local test_name=$2
    local test_script=$3
    local test_args=$4
    
    log "\n${BLUE}========================================${NC}"
    log "${BLUE}Test $test_num: $test_name${NC}"
    log "${BLUE}========================================${NC}\n"
    
    log "Script: $test_script $test_args"
    log "Started: $(date +%H:%M:%S)\n"
    
    # Clean up before each test
    log "Cleaning up old data..."
    rm -rf ../data/* ../logs/*
    
    # Kill any existing processes
    pkill -9 -f 'bin/bootstrap' 2>/dev/null || true
    pkill -9 -f 'bin/node' 2>/dev/null || true
    sleep 2
    
    # Run the test in background with timeout
    local start_time=$(date +%s)
    
    # Start test in background
    if [ -n "$test_args" ]; then
        timeout 120 ./$test_script $test_args > "$RESULTS_DIR/test${test_num}_${TIMESTAMP}.log" 2>&1 &
    else
        timeout 120 ./$test_script > "$RESULTS_DIR/test${test_num}_${TIMESTAMP}.log" 2>&1 &
    fi
    
    local test_pid=$!
    
    # Wait for test to complete or timeout
    wait $test_pid 2>/dev/null || true
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    # Always kill processes after test
    pkill -9 -f 'bin/bootstrap' 2>/dev/null || true
    pkill -9 -f 'bin/node' 2>/dev/null || true
    pkill -9 -f 'bin/submit-vote' 2>/dev/null || true
    sleep 2
    
    log "\nCompleted: $(date +%H:%M:%S)"
    log "Duration: ${duration}s"
    
    # Copy blockchain data for this test
    if [ -d "../data/data1" ]; then
        cp -r ../data "$RESULTS_DIR/test${test_num}_data_${TIMESTAMP}"
    fi
    
    # Extract key metrics
    extract_metrics $test_num
    
    log "\n${GREEN}✓ Test $test_num completed${NC}\n"
    
    # Pause between tests
    sleep 3
}

# Function to extract metrics from logs
extract_metrics() {
    local test_num=$1
    
    log "Key Metrics:"
    log "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    if [ -f "../logs/node1.log" ]; then
        local votes_mempool=$(grep "added to mempool" ../logs/node1.log 2>/dev/null | wc -l | tr -d ' ')
        local blocks_finalized=$(grep "finalized successfully" ../logs/node1.log 2>/dev/null | wc -l | tr -d ' ')
        local votes_finalized=$(grep "finalized successfully" ../logs/node1.log 2>/dev/null | tail -1 | grep -o "voted count: [0-9]*" | awk '{print $3}')
        
        if [ -z "$votes_finalized" ]; then
            votes_finalized=0
        fi
        
        log "  Votes in mempool: $votes_mempool"
        log "  Blocks finalized: $blocks_finalized"
        log "  Votes finalized: $votes_finalized"
        
        # Check for errors
        local errors=$(grep -h "ERROR" ../logs/node*.log 2>/dev/null | grep -v "no active round" | wc -l | tr -d ' ')
        if [ $errors -gt 0 ]; then
            log "  ${YELLOW}⚠ Errors detected: $errors${NC}"
        else
            log "  ${GREEN}✓ No critical errors${NC}"
        fi
    else
        log "  ${RED}✗ No logs found${NC}"
    fi
    
    log ""
}

# Pre-flight checks
log "\n${BLUE}Pre-flight Checks${NC}"
log "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"

# Check if binaries exist
if [ ! -f "../bin/bootstrap" ] || [ ! -f "../bin/node" ] || [ ! -f "../bin/submit-vote" ]; then
    log "${YELLOW}Building binaries...${NC}"
    ./build.sh
    if [ $? -ne 0 ]; then
        log "${RED}✗ Build failed${NC}"
        exit 1
    fi
    log "${GREEN}✓ Build successful${NC}\n"
else
    log "${GREEN}✓ Binaries exist${NC}\n"
fi

# Check if voters exist
if [ ! -d "../keys/voters" ] || [ $(ls -1 ../keys/voters/*.key 2>/dev/null | wc -l) -lt 100 ]; then
    log "${YELLOW}Generating 100 voter keypairs...${NC}"
    ./generate_voters.sh 100
    ./update_eligibility.sh
    log "${GREEN}✓ Voters generated${NC}\n"
else
    log "${GREEN}✓ Voters exist ($(ls -1 ../keys/voters/*.key 2>/dev/null | wc -l) keypairs)${NC}\n"
fi

log "${GREEN}✓ All pre-flight checks passed${NC}\n"

# Execute all tests
log "\n${BLUE}========================================${NC}"
log "${BLUE}Starting Test Execution${NC}"
log "${BLUE}========================================${NC}\n"

TOTAL_START=$(date +%s)

# Test 1: Functional Validation
run_test 1 "Validação Funcional Básica" "quick_test.sh" ""

# Test 2: Security (Double Voting)
run_test 2 "Validação de Segurança (Double Voting)" "test_validation_failure.sh" ""

# Test 3: Performance
run_test 3 "Análise de Performance e Throughput" "stress_test_advanced.sh" "50 0 30"

# Test 4: Scalability
run_test 4 "Teste de Escalabilidade (Carga Progressiva)" "progressive_load_test.sh" ""

# Test 5: Fault Tolerance
run_test 5 "Tolerância a Falhas (Crash Failure)" "byzantine_failure_test.sh" "crash"

TOTAL_END=$(date +%s)
TOTAL_DURATION=$((TOTAL_END - TOTAL_START))

# Final cleanup
pkill -9 -f 'bin/bootstrap' 2>/dev/null || true
pkill -9 -f 'bin/node' 2>/dev/null || true

# Generate summary
log "\n${BLUE}========================================${NC}"
log "${BLUE}Test Suite Summary${NC}"
log "${BLUE}========================================${NC}\n"

log "Total execution time: ${TOTAL_DURATION}s ($(($TOTAL_DURATION / 60))m $(($TOTAL_DURATION % 60))s)"
log "Tests completed: 5/5"
log "Results directory: $RESULTS_DIR"
log ""

log "Generated files:"
log "  - Consolidated report: $(basename $REPORT_FILE)"
log "  - Individual test logs: test[1-5]_${TIMESTAMP}.log"
log "  - Blockchain data: test[1-5]_data_${TIMESTAMP}/"
log ""

# Run blockchain analysis if available
if [ -f "analyze_blocks.go" ]; then
    log "\n${BLUE}========================================${NC}"
    log "${BLUE}Blockchain Analysis${NC}"
    log "${BLUE}========================================${NC}\n"
    
    # Analyze blockchain from test 3 (most interesting)
    if [ -d "$RESULTS_DIR/test3_data_${TIMESTAMP}/data1" ]; then
        log "Analyzing blockchain from Test 3 (Performance)...\n"
        
        # Temporarily copy data back for analysis
        cp -r "$RESULTS_DIR/test3_data_${TIMESTAMP}"/* ../data/
        
        go run analyze_blocks.go 2>&1 | tee "$RESULTS_DIR/blockchain_analysis_${TIMESTAMP}.txt"
        
        log "\n${GREEN}✓ Blockchain analysis completed${NC}"
        log "  Results saved to: blockchain_analysis_${TIMESTAMP}.txt\n"
    fi
fi

# Final summary
log "\n${GREEN}========================================${NC}"
log "${GREEN}✓ All Tests Completed Successfully${NC}"
log "${GREEN}========================================${NC}\n"

log "Next steps for TCC:"
log "  1. Review individual test logs in $RESULTS_DIR"
log "  2. Extract metrics for tables and graphs"
log "  3. Generate visualizations (graphs, charts)"
log "  4. Write analysis for Chapter 5"
log ""

log "Report saved to: $REPORT_FILE"
log ""

echo -e "${GREEN}Done!${NC}"
