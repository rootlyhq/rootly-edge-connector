#!/bin/sh
# Pre-removal script for RPM/DEB packages
# Stops and disables the service before package removal

set -e

if command -v systemctl >/dev/null 2>&1; then
    # Stop the service if it's running
    if systemctl is-active --quiet rootly-edge-connector; then
        echo "Stopping rootly-edge-connector service..."
        systemctl stop rootly-edge-connector || true
    fi

    # Disable the service
    if systemctl is-enabled --quiet rootly-edge-connector; then
        echo "Disabling rootly-edge-connector service..."
        systemctl disable rootly-edge-connector || true
    fi
fi

exit 0
