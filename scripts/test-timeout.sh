#!/bin/bash
# Test script that times out
# Use this to test timeout handling

set -e

SLEEP_DURATION="${REC_PARAM_SLEEP_DURATION:-60}"

echo "Starting long-running test script..."
echo "Will sleep for $SLEEP_DURATION seconds"
echo "Configure action timeout to be less than this to test timeout handling"
echo ""

for i in $(seq 1 $SLEEP_DURATION); do
    echo "Progress: $i/$SLEEP_DURATION seconds..."
    sleep 1
done

echo ""
echo "Script completed (this shouldn't print if timeout is working)"

exit 0
