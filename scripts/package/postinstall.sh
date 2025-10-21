#!/bin/sh
# Post-installation script for RPM/DEB packages
# Sets up permissions, directories, and systemd service

set -e

# Create necessary directories if they don't exist
mkdir -p /opt/rootly-edge-connector/bin
mkdir -p /opt/rootly-edge-connector/scripts
mkdir -p /etc/rootly-edge-connector
mkdir -p /var/log/rootly-edge-connector

# Set ownership
chown -R rootly:rootly /opt/rootly-edge-connector
chown -R rootly:rootly /etc/rootly-edge-connector
chown -R rootly:rootly /var/log/rootly-edge-connector

# Set permissions on config directory
chmod 755 /etc/rootly-edge-connector
chmod 644 /etc/rootly-edge-connector/*.yml 2>/dev/null || true

# Set binary permissions
chmod 755 /usr/bin/rootly-edge-connector

# Reload systemd daemon to pick up service file
if command -v systemctl >/dev/null 2>&1; then
    systemctl daemon-reload

    # Enable service (but don't start it automatically)
    # Users should configure API key first before starting
    systemctl enable rootly-edge-connector || true

    echo ""
    echo "Rootly Edge Connector installed successfully!"
    echo ""
    echo "Next steps:"
    echo "  1. Copy example configs: cp /etc/rootly-edge-connector/config.example.yml /etc/rootly-edge-connector/config.yml"
    echo "  2. Copy example configs: cp /etc/rootly-edge-connector/actions.example.yml /etc/rootly-edge-connector/actions.yml"
    echo "  3. Edit config and add your API key"
    echo "  4. Start the service: sudo systemctl start rootly-edge-connector"
    echo "  5. Check status: sudo systemctl status rootly-edge-connector"
    echo ""
fi

exit 0
