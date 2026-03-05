#!/usr/bin/env python3
"""
SBFL (Statistical-Based Fault Localization) Analyzer
Analyzes test failures to identify the most suspicious files
"""

import re
import json
import sys
from collections import defaultdict

def parse_test_output(test_log):
    """Parse go test output to find failing tests and their locations"""
    suspicious = defaultdict(int)
    
    # Read test log
    try:
        with open(test_log) as f:
            content = f.read()
    except:
        content = test_log  # Input might be the content itself
    
    # Pattern 1: FAIL lines with package names
    for match in re.finditer(r'^FAIL\s+(\S+)', content, re.MULTILINE):
        pkg = match.group(1)
        suspicious[pkg] += 3
        # Also mark related files
        for suffix in ['.go', '_test.go']:
            suspicious[f"{pkg}/*{suffix}"] += 2
    
    # Pattern 2: Error messages with file:line
    for match in re.finditer(r'(\S+\.go):(\d+):\s+(.+)', content):
        file = match.group(1)
        line = match.group(2)
        error = match.group(3)
        
        # Weight by error type
        weight = 1
        if 'nil pointer' in error.lower():
            weight = 5
        elif 'type' in error.lower() and 'error' in error.lower():
            weight = 4
        elif 'undefined' in error.lower():
            weight = 3
        
        suspicious[file] += weight
    
    # Pattern 3: panic messages with goroutine stack traces
    for match in re.finditer(r'panic: (.+?)\n.+?([\w/]+\.go):(\d+)', content, re.DOTALL):
        file = match.group(2)
        line = match.group(3)
        suspicious[file] += 5  # Panics are critical
    
    # Pattern 4: Test names that failed
    for match in re.finditer(r'--- FAIL: (\S+)', content):
        test_name = match.group(1)
        # Extract function name from test
        func_match = re.match(r'Test(\w+)', test_name)
        if func_match:
            func_name = func_match.group(1)
            # Convert test name to potential file
            file_name = func_name.lower().replace('_', '')
            suspicious[f"*_test.go (Test{func_name})"] += 3
    
    return dict(suspicious)

def main():
    if len(sys.argv) < 3:
        print(json.dumps({"error": "Usage: sbfl_analyzer.py <test_log> <output_json>"}))
        sys.exit(1)
    
    test_log = sys.argv[1]
    output_json = sys.argv[2]
    
    # Analyze
    suspicious = parse_test_output(test_log)
    
    # Sort by suspicion score
    ranked = sorted(suspicious.items(), key=lambda x: x[1], reverse=True)
    
    # Create output
    result = {
        "total_files": len(ranked),
        "top_suspicious": [
            {"file": f, "score": s, "rank": i+1}
            for i, (f, s) in enumerate(ranked[:20])
        ],
        "recommendation": ranked[0][0] if ranked else "No clear suspect"
    }
    
    # Save
    with open(output_json, 'w') as f:
        json.dump(result, f, indent=2)
    
    # Print summary
    print(f"  📊 SBFL Analysis: {len(ranked)} suspicious files found")
    if ranked:
        print(f"  🎯 Most suspicious: {ranked[0][0]} (score: {ranked[0][1]})")
    
if __name__ == '__main__':
    main()
