#!/bin/bash
# Test script that simulates a failure
# Use this to test error handling and reporting

set -e

echo "Starting test script that will fail..."
echo "Event ID: ${REC_PARAM_EVENT_ID:-not provided}"
echo ""

# Simulate some work
sleep 1

# Simulate a failure
echo "ERROR: Simulated failure condition detected!" >&2
echo "This script is designed to fail for testing purposes" >&2

exit 1
