#!/bin/sh
# Post-removal script for RPM/DEB packages
# Cleans up after package removal (optional - can leave user/data for reinstall)

set -e

# Reload systemd daemon to remove service file
if command -v systemctl >/dev/null 2>&1; then
    systemctl daemon-reload || true
fi

# Note: We intentionally DO NOT remove:
# - /etc/rootly-edge-connector (user may want to keep configs)
# - /var/log/rootly-edge-connector (preserve logs)
# - rootly user/group (may be used for reinstall)
#
# To completely remove everything:
#   sudo userdel rootly
#   sudo groupdel rootly
#   sudo rm -rf /etc/rootly-edge-connector
#   sudo rm -rf /var/log/rootly-edge-connector
#   sudo rm -rf /opt/rootly-edge-connector

echo "Rootly Edge Connector removed."
echo "Note: Configuration (/etc/rootly-edge-connector) and logs (/var/log/rootly-edge-connector) were preserved."
echo "To completely remove: sudo rm -rf /etc/rootly-edge-connector /var/log/rootly-edge-connector /opt/rootly-edge-connector"

exit 0
