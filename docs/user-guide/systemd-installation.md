# Systemd Installation Guide

## Prerequisites

- Linux system with systemd
- Root or sudo access

## Installation Steps

### 1. Create User and Group

```bash
sudo groupadd -r rootly
sudo useradd -r -g rootly -s /bin/false -d /opt/rootly-edge-connector rootly
```

### 2. Create Directories

```bash
sudo mkdir -p /opt/rootly-edge-connector/bin
sudo mkdir -p /opt/rootly-edge-connector/scripts
sudo mkdir -p /etc/rootly-edge-connector
sudo mkdir -p /var/log/rootly-edge-connector
```

### 3. Install Binary

```bash
# Copy the binary
sudo cp bin/rootly-edge-connector /opt/rootly-edge-connector/bin/
sudo chmod +x /opt/rootly-edge-connector/bin/rootly-edge-connector
```

### 4. Install Configuration

```bash
# Copy configuration files
sudo cp config.yml /etc/rootly-edge-connector/
sudo cp actions.yml /etc/rootly-edge-connector/

# Create environment file with API key
sudo tee /etc/rootly-edge-connector/environment > /dev/null <<EOF
REC_API_KEY=your-actual-api-key-here
EOF
```

### 5. Set Permissions

```bash
sudo chown -R rootly:rootly /opt/rootly-edge-connector
sudo chown -R rootly:rootly /etc/rootly-edge-connector
sudo chown -R rootly:rootly /var/log/rootly-edge-connector

# Protect sensitive files
sudo chmod 600 /etc/rootly-edge-connector/environment
sudo chmod 640 /etc/rootly-edge-connector/config.yml
sudo chmod 640 /etc/rootly-edge-connector/actions.yml
```

### 6. Install Systemd Service

```bash
sudo cp docs/rootly-edge-connector.service /etc/systemd/system/
sudo systemctl daemon-reload
```

### 7. Enable and Start Service

```bash
# Enable service to start on boot
sudo systemctl enable rootly-edge-connector

# Start service
sudo systemctl start rootly-edge-connector

# Check status
sudo systemctl status rootly-edge-connector
```

## Management Commands

```bash
# Start service
sudo systemctl start rootly-edge-connector

# Stop service
sudo systemctl stop rootly-edge-connector

# Restart service
sudo systemctl restart rootly-edge-connector

# View status
sudo systemctl status rootly-edge-connector

# View logs
sudo journalctl -u rootly-edge-connector -f

# View recent logs
sudo journalctl -u rootly-edge-connector -n 100
```

## Accessing Metrics

If metrics are enabled (default port 9090):

```bash
curl http://localhost:9090/metrics
```

## Updating

### Binary Update

```bash
# Stop service
sudo systemctl stop rootly-edge-connector

# Replace binary
sudo cp new-binary /opt/rootly-edge-connector/bin/rootly-edge-connector
sudo chmod +x /opt/rootly-edge-connector/bin/rootly-edge-connector
sudo chown rootly:rootly /opt/rootly-edge-connector/bin/rootly-edge-connector

# Start service
sudo systemctl start rootly-edge-connector
```

### Configuration Update

```bash
# Edit config
sudo vim /etc/rootly-edge-connector/config.yml
sudo vim /etc/rootly-edge-connector/actions.yml

# Reload service
sudo systemctl restart rootly-edge-connector
```

## Troubleshooting

### Check Service Status

```bash
sudo systemctl status rootly-edge-connector
```

### View Logs

```bash
# Follow logs in real-time
sudo journalctl -u rootly-edge-connector -f

# Show last 100 lines
sudo journalctl -u rootly-edge-connector -n 100

# Show errors only
sudo journalctl -u rootly-edge-connector -p err
```

### Common Issues

**Service won't start:**
```bash
# Check configuration
/opt/rootly-edge-connector/bin/rootly-edge-connector \
    -config /etc/rootly-edge-connector/config.yml \
    -actions /etc/rootly-edge-connector/actions.yml

# Check permissions
ls -la /opt/rootly-edge-connector/
ls -la /etc/rootly-edge-connector/
```

**API key issues:**
```bash
# Verify environment file
sudo cat /etc/rootly-edge-connector/environment

# Test with environment variable
sudo -u rootly REC_API_KEY="your-key" \
    /opt/rootly-edge-connector/bin/rootly-edge-connector \
    -config /etc/rootly-edge-connector/config.yml \
    -actions /etc/rootly-edge-connector/actions.yml
```

## Uninstallation

```bash
# Stop and disable service
sudo systemctl stop rootly-edge-connector
sudo systemctl disable rootly-edge-connector

# Remove service file
sudo rm /etc/systemd/system/rootly-edge-connector.service
sudo systemctl daemon-reload

# Remove files (optional)
sudo rm -rf /opt/rootly-edge-connector
sudo rm -rf /etc/rootly-edge-connector
sudo rm -rf /var/log/rootly-edge-connector

# Remove user (optional)
sudo userdel rootly
sudo groupdel rootly
```
