#!/bin/bash
# Test script demonstrating conditional logic
# Shows how to handle different severities or conditions

set -e

SEVERITY="${REC_PARAM_SEVERITY:-unknown}"
EVENT_TYPE="${REC_PARAM_EVENT_TYPE:-unknown}"

echo "========================================"
echo "Conditional Test Script"
echo "========================================"
echo ""
echo "Event Type: $EVENT_TYPE"
echo "Severity: $SEVERITY"
echo ""

# Conditional logic based on severity
case "$SEVERITY" in
    critical)
        echo "🚨 CRITICAL severity detected!"
        echo "Action: Would trigger emergency response"
        echo "Action: Would page on-call team"
        echo "Action: Would scale up resources"
        ;;
    high)
        echo "⚠️  HIGH severity detected"
        echo "Action: Would notify team"
        echo "Action: Would increase monitoring"
        ;;
    medium)
        echo "ℹ️  MEDIUM severity detected"
        echo "Action: Would log for review"
        ;;
    low)
        echo "✓ LOW severity detected"
        echo "Action: Would track in metrics"
        ;;
    *)
        echo "Unknown severity: $SEVERITY"
        echo "Action: Would handle with default policy"
        ;;
esac

echo ""
echo "========================================"
echo "Conditional logic completed successfully"
echo "========================================"

exit 0
