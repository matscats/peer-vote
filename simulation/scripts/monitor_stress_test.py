#!/usr/bin/env python3
"""
Real-time monitoring for stress test
Tracks votes, blocks, and performance metrics
"""

import os
import sys
import time
import re
from collections import defaultdict
from datetime import datetime

class StressTestMonitor:
    def __init__(self, logs_dir="../logs"):
        self.logs_dir = logs_dir
        self.start_time = time.time()
        self.metrics = {
            'votes_submitted': 0,
            'votes_in_mempool': 0,
            'blocks_finalized': 0,
            'total_votes_finalized': 0,
            'blocks_by_node': defaultdict(int),
            'votes_by_candidate': defaultdict(int),
            'last_block_time': None,
            'block_times': [],
        }
        
    def parse_logs(self):
        """Parse all log files and update metrics"""
        # Parse vote submission logs
        vote_files = [f for f in os.listdir(self.logs_dir) if f.startswith('vote_') and f.endswith('.log')]
        self.metrics['votes_submitted'] = len([
            f for f in vote_files 
            if self._file_contains(os.path.join(self.logs_dir, f), 'Vote submitted successfully')
        ])
        
        # Parse vote distribution
        for vote_file in vote_files:
            content = self._read_file(os.path.join(self.logs_dir, vote_file))
            match = re.search(r'choice=([a-zA-Z0-9_-]+)', content)
            if match:
                self.metrics['votes_by_candidate'][match.group(1)] += 1
        
        # Parse node1 log for blockchain state
        node1_log = os.path.join(self.logs_dir, 'node1.log')
        if os.path.exists(node1_log):
            content = self._read_file(node1_log)
            
            # Count blocks finalized
            finalized_lines = re.findall(r'Block at height (\d+) finalized successfully.*voted count: (\d+)', content)
            if finalized_lines:
                self.metrics['blocks_finalized'] = len(finalized_lines)
                last_height, last_votes = finalized_lines[-1]
                self.metrics['total_votes_finalized'] = int(last_votes)
            
            # Count mempool size
            mempool_lines = re.findall(r'mempool size: (\d+)', content)
            if mempool_lines:
                self.metrics['votes_in_mempool'] = int(mempool_lines[-1])
            
            # Calculate block times
            block_times = re.findall(r'(\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}).*Block at height \d+ finalized', content)
            if len(block_times) > 1:
                times = [datetime.strptime(t, '%Y/%m/%d %H:%M:%S') for t in block_times]
                intervals = [(times[i+1] - times[i]).total_seconds() for i in range(len(times)-1)]
                self.metrics['block_times'] = intervals
                self.metrics['last_block_time'] = times[-1]
        
        # Parse blocks proposed by each node
        for i in range(1, 4):
            node_log = os.path.join(self.logs_dir, f'node{i}.log')
            if os.path.exists(node_log):
                content = self._read_file(node_log)
                proposed = len(re.findall(r'Proposed new block at height', content))
                self.metrics['blocks_by_node'][f'node{i}'] = proposed
    
    def _read_file(self, filepath):
        """Read file content safely"""
        try:
            with open(filepath, 'r') as f:
                return f.read()
        except:
            return ""
    
    def _file_contains(self, filepath, text):
        """Check if file contains text"""
        try:
            with open(filepath, 'r') as f:
                return text in f.read()
        except:
            return False
    
    def calculate_metrics(self):
        """Calculate derived metrics"""
        elapsed = time.time() - self.start_time
        
        throughput = self.metrics['total_votes_finalized'] / elapsed if elapsed > 0 else 0
        
        avg_block_time = sum(self.metrics['block_times']) / len(self.metrics['block_times']) if self.metrics['block_times'] else 0
        
        vote_loss = self.metrics['votes_submitted'] - self.metrics['total_votes_finalized']
        vote_loss_pct = (vote_loss / self.metrics['votes_submitted'] * 100) if self.metrics['votes_submitted'] > 0 else 0
        
        avg_votes_per_block = self.metrics['total_votes_finalized'] / self.metrics['blocks_finalized'] if self.metrics['blocks_finalized'] > 0 else 0
        
        return {
            'elapsed': elapsed,
            'throughput': throughput,
            'avg_block_time': avg_block_time,
            'vote_loss': vote_loss,
            'vote_loss_pct': vote_loss_pct,
            'avg_votes_per_block': avg_votes_per_block,
        }
    
    def display(self):
        """Display current metrics"""
        self.parse_logs()
        derived = self.calculate_metrics()
        
        # Clear screen
        os.system('clear' if os.name == 'posix' else 'cls')
        
        print("=" * 60)
        print("🔥 STRESS TEST MONITOR - Real-time Metrics")
        print("=" * 60)
        print()
        
        print(f"⏱️  Elapsed Time: {derived['elapsed']:.1f}s")
        print()
        
        print("📊 Vote Submission:")
        print(f"  Submitted: {self.metrics['votes_submitted']}")
        print(f"  In Mempool: {self.metrics['votes_in_mempool']}")
        print(f"  Finalized: {self.metrics['total_votes_finalized']}")
        print(f"  Lost: {derived['vote_loss']} ({derived['vote_loss_pct']:.1f}%)")
        print()
        
        print("📦 Blockchain:")
        print(f"  Blocks Finalized: {self.metrics['blocks_finalized']}")
        print(f"  Avg Votes/Block: {derived['avg_votes_per_block']:.2f}")
        print(f"  Avg Block Time: {derived['avg_block_time']:.2f}s")
        print()
        
        print("🚀 Performance:")
        print(f"  Throughput: {derived['throughput']:.2f} votes/sec")
        print()
        
        print("🗳️  Vote Distribution:")
        for candidate, count in sorted(self.metrics['votes_by_candidate'].items()):
            bar = "█" * (count // 2)
            print(f"  {candidate:15s}: {count:3d} {bar}")
        print()
        
        print("🔧 Block Proposals by Node:")
        for node, count in sorted(self.metrics['blocks_by_node'].items()):
            print(f"  {node}: {count}")
        print()
        
        # Status indicator
        if derived['vote_loss'] == 0 and self.metrics['total_votes_finalized'] > 0:
            print("✅ Status: EXCELLENT - No vote loss")
        elif derived['vote_loss_pct'] < 5:
            print("✅ Status: GOOD - Minimal vote loss")
        elif derived['vote_loss_pct'] < 10:
            print("⚠️  Status: ACCEPTABLE - Some vote loss")
        else:
            print("❌ Status: POOR - Significant vote loss")
        
        print()
        print("Press Ctrl+C to stop monitoring")
        print("=" * 60)
    
    def run(self, interval=2):
        """Run monitoring loop"""
        try:
            while True:
                self.display()
                time.sleep(interval)
        except KeyboardInterrupt:
            print("\n\nMonitoring stopped.")
            print("\nFinal Results:")
            self.display()

if __name__ == "__main__":
    monitor = StressTestMonitor()
    monitor.run(interval=2)
