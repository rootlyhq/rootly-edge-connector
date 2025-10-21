#!/bin/sh
# Pre-installation script for RPM/DEB packages
# Creates the rootly user and group if they don't exist

set -e

# Create rootly group if it doesn't exist
if ! getent group rootly >/dev/null 2>&1; then
    echo "Creating rootly group..."
    groupadd -r rootly
fi

# Create rootly user if it doesn't exist
if ! getent passwd rootly >/dev/null 2>&1; then
    echo "Creating rootly user..."
    useradd -r -g rootly -s /bin/false -d /opt/rootly-edge-connector -c "Rootly Edge Connector" rootly
fi

exit 0
