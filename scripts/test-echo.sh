#!/bin/bash
# Simple test script for local development
# This script just echoes parameters and environment info

set -e

echo "========================================"
echo "Rootly Edge Connector - Test Script"
echo "========================================"
echo ""
echo "Timestamp: $(date)"
echo "Hostname: $(hostname)"
echo ""
echo "--- Event Parameters ---"
echo "Event ID: ${REC_PARAM_EVENT_ID:-not provided}"
echo "Event Type: ${REC_PARAM_EVENT_TYPE:-not provided}"
echo "Host: ${REC_PARAM_HOST:-not provided}"
echo "Service: ${REC_PARAM_SERVICE:-not provided}"
echo "Severity: ${REC_PARAM_SEVERITY:-not provided}"
echo ""
echo "--- Environment Variables ---"
echo "ENVIRONMENT: ${ENVIRONMENT:-not set}"
echo "LOG_LEVEL: ${LOG_LEVEL:-not set}"
echo ""
echo "--- All REC_PARAM_* Variables ---"
env | grep "^REC_PARAM_" || echo "(none)"
echo ""
echo "========================================"
echo "Test script completed successfully!"
echo "========================================"

exit 0
