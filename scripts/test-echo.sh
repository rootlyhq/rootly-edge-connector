#!/bin/bash
# Simple test script for local development
# This script just echoes parameters and environment info
#
# Expected parameters (passed as REC_PARAM_* environment variables):
#   alert_id    -> REC_PARAM_ALERT_ID
#   summary     -> REC_PARAM_SUMMARY
#   host        -> REC_PARAM_HOST
#   service     -> REC_PARAM_SERVICE
#   severity    -> REC_PARAM_SEVERITY
#
# Usage in actions.yml:
#   parameters:
#     alert_id: "{{ id }}"
#     summary: "{{ summary }}"
#     host: "{{ data.host }}"
#     service: "{{ services[0].name }}"
#     severity: "{{ labels.severity }}"

set -e

echo "========================================"
echo "Rootly Edge Connector - Test Script"
echo "========================================"
echo ""
echo "Timestamp: $(date)"
echo "Hostname: $(hostname)"
echo ""
echo "--- Event Parameters ---"
echo "Alert ID: ${REC_PARAM_ALERT_ID:-not provided}"
echo "Summary: ${REC_PARAM_SUMMARY:-not provided}"
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
