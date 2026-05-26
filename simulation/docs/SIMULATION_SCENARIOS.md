# Simulation Scenarios for TCC

This document describes various election simulation scenarios you can run to generate data and insights for your thesis.

## 📊 Basic Scenarios

### Scenario 1: Small Election (10 voters)

**Purpose**: Demonstrate basic functionality

```bash
./simulate_election.sh 10 2
```

**Expected Results**:
- All 10 votes should be finalized
- 2-4 blocks depending on timing
- ~30 seconds total duration

**Metrics to Track**:
- Time to first block
- Average votes per block
- Total finalization time

---

### Scenario 2: Medium Election (50 voters)

**Purpose**: Test throughput and performance

```bash
./simulate_election.sh 50 1
```

**Expected Results**:
- 50 votes across 10-12 blocks
- ~60 seconds total duration
- Consistent block intervals

**Metrics to Track**:
- Votes per second
- Block finalization latency
- Network message count

---

### Scenario 3: Large Election (100 voters)

**Purpose**: Stress test the system

```bash
./simulate_election.sh 100 0.5
```

**Expected Results**:
- 100 votes across 20-25 blocks
- ~90 seconds total duration
- Possible mempool backlog

**Metrics to Track**:
- Maximum mempool size
- Block propagation time
- Consensus round duration

---

## 🔬 Advanced Scenarios

### Scenario 4: Rapid Voting

**Purpose**: Test system under high load

```bash
# Submit 30 votes with 0.2s intervals
./simulate_election.sh 30 0.2
```

**What to Observe**:
- Mempool behavior
- Vote batching in blocks
- Network congestion

---

### Scenario 5: Distributed Voting

**Purpose**: Simulate realistic voting patterns

**Setup**: Modify `simulate_election.sh` to add delays between voter groups

```bash
# Group 1: 20 voters (morning)
./simulate_election.sh 20 1

# Wait 30 seconds

# Group 2: 30 voters (afternoon)
# Continue submitting votes to running network
```

**What to Observe**:
- Block density variation
- Idle periods between voting waves
- State consistency

---

### Scenario 6: Close Election

**Purpose**: Test with evenly distributed votes

**Setup**: Modify candidate selection to be more balanced

```bash
# Edit simulate_election.sh to use weighted random selection
# Ensure ~33% for each candidate
./simulate_election.sh 60 1
```

**What to Observe**:
- Vote distribution accuracy
- Final tally correctness
- Blockchain immutability

---

## 🛠️ Custom Scenarios

### Creating Your Own Scenario

1. **Copy the base script**:
   ```bash
   cp simulate_election.sh my_scenario.sh
   ```

2. **Modify parameters**:
   - Number of voters
   - Vote intervals
   - Candidate selection logic
   - Network delays

3. **Run and document**:
   ```bash
   ./my_scenario.sh > results.txt 2>&1
   ```

---

## 📈 Data Collection

### For Each Scenario, Collect:

#### Performance Metrics
```bash
# Block finalization times
grep "finalized successfully" ../logs/node*.log | \
  awk '{print $1, $2}' | \
  uniq -c

# Vote throughput
VOTES=$(grep "voted count:" ../logs/node*.log | awk '{sum += $NF} END {print sum}')
DURATION=60  # seconds
echo "Throughput: $(($VOTES / $DURATION)) votes/sec"
```

#### Network Metrics
```bash
# Peer connections over time
grep "Connected peers:" ../logs/node*.log | \
  awk '{print $1, $2, $NF}'

# DHT discoveries
grep "DHT discovered peer" ../logs/node*.log | wc -l
```

#### Consensus Metrics
```bash
# Blocks per validator
for i in 1 2 3; do
  echo "Node $i proposed:"
  grep "Proposed new block" ../logs/node${i}.log | wc -l
done

# Approval counts
grep "Recorded approval" ../logs/node*.log | wc -l
```

---

## 🎯 Scenarios for Specific TCC Sections

### For "System Architecture" Section

**Run**: Basic 10-voter scenario
**Capture**: 
- Network topology (peer connections)
- Component interaction (logs showing event flow)
- State transitions (consensus states)

### For "Performance Analysis" Section

**Run**: Multiple scenarios with varying loads
**Capture**:
- Throughput graphs (votes/sec vs. load)
- Latency measurements (proposal to finalization)
- Resource usage (if monitoring tools available)

### For "Consensus Validation" Section

**Run**: Scenarios with different validator counts
**Capture**:
- Byzantine fault tolerance demonstration
- Leader rotation verification
- Finality guarantees

### For "Security Analysis" Section

**Run**: Scenarios with invalid votes
**Capture**:
- Rejection of duplicate votes
- Signature verification
- Eligibility checking

---

## 📊 Visualization Ideas

### Vote Distribution Chart
```bash
# Generate data for pie chart
./analyze_results.sh | grep "candidate-" | \
  awk '{print $1, $2}'
```

### Block Timeline
```bash
# Generate data for timeline
grep "finalized successfully" ../logs/node1.log | \
  awk '{print NR, $1, $2, $8}' > block_timeline.csv
```

### Network Growth
```bash
# Track peer connections over time
grep "Connected peers:" ../logs/node*.log | \
  awk '{print $1, $2, $NF}' > network_growth.csv
```

---

## 🔍 Troubleshooting Scenarios

### Scenario Fails to Complete

**Check**:
1. All nodes are running: `ps aux | grep bin/node`
2. Bootstrap node is accessible
3. Logs for errors: `grep ERROR ../logs/*.log`

### Votes Not Being Finalized

**Check**:
1. Votes are being submitted: `ls ../logs/vote_*.log`
2. Votes reach mempool: `grep "added to mempool" ../logs/*.log`
3. Blocks are being proposed: `grep "Proposed new block" ../logs/*.log`

### Inconsistent Results

**Solution**:
1. Clean all data: `rm -rf ../data/* ../logs/*`
2. Restart all nodes
3. Re-run scenario

---

## 📝 Documentation Template

For each scenario in your TCC, document:

```markdown
### Scenario X: [Name]

**Objective**: [What you're testing]

**Setup**:
- Validators: 3
- Voters: [N]
- Vote interval: [X]s
- Duration: [Y]s

**Execution**:
[Command used]

**Results**:
- Blocks finalized: [N]
- Votes processed: [N]
- Average latency: [X]ms
- Throughput: [X] votes/sec

**Observations**:
[Key findings]

**Graphs**:
[Reference to figures in thesis]
```

---

## 🎓 Tips for TCC

1. **Run each scenario 3 times** for statistical validity
2. **Document everything** - logs, configs, commands
3. **Take screenshots** of monitor output
4. **Save raw data** for later analysis
5. **Compare scenarios** to show system behavior under different conditions

Good luck with your thesis! 🎉
