# Package Scripts

These scripts are executed during RPM/DEB package installation and removal lifecycle.

## Scripts

### `preinstall.sh`
- **When**: Before package files are installed
- **Purpose**: Create rootly user and group
- **Actions**:
  - Creates `rootly` system group if it doesn't exist
  - Creates `rootly` system user with home directory `/opt/rootly-edge-connector`

### `postinstall.sh`
- **When**: After package files are installed
- **Purpose**: Set up permissions and enable service
- **Actions**:
  - Creates necessary directories (`/opt/rootly-edge-connector`, `/etc/rootly-edge-connector`, `/var/log/rootly-edge-connector`)
  - Sets ownership to `rootly:rootly`
  - Sets proper permissions on configs and binary
  - Reloads systemd daemon
  - Enables service (but doesn't start it - user must configure API key first)
  - Displays setup instructions

### `preremove.sh`
- **When**: Before package files are removed
- **Purpose**: Stop the running service
- **Actions**:
  - Stops rootly-edge-connector service if running
  - Disables service from starting on boot

### `postremove.sh`
- **When**: After package files are removed
- **Purpose**: Clean up systemd state
- **Actions**:
  - Reloads systemd daemon to remove service
  - Preserves config files, logs, and user/group for potential reinstall
  - Displays instructions for complete removal

## Usage in GoReleaser

These scripts are referenced in `.goreleaser.yml`:

```yaml
nfpms:
  - scripts:
      preinstall: scripts/package/preinstall.sh
      postinstall: scripts/package/postinstall.sh
      preremove: scripts/package/preremove.sh
      postremove: scripts/package/postremove.sh
```

## Testing Package Scripts

To test the scripts manually:

```bash
# Install package
sudo rpm -ivh rootly-edge-connector-1.0.0.x86_64.rpm
# or
sudo dpkg -i rootly-edge-connector_1.0.0_amd64.deb

# Check service status
sudo systemctl status rootly-edge-connector

# Remove package
sudo rpm -e rootly-edge-connector
# or
sudo dpkg -r rootly-edge-connector
```

## Design Decisions

### User/Group Preservation
The user and group are **NOT** removed during package removal. This is intentional:
- Allows for seamless package upgrades
- Preserves file ownership if configs/logs remain
- Follows common practice for system services

### Config/Log Preservation
Configuration files and logs are **NOT** removed during package removal:
- `/etc/rootly-edge-connector/` - Preserved (user may want to reinstall)
- `/var/log/rootly-edge-connector/` - Preserved (important for troubleshooting)

Users can manually remove these if desired after uninstallation.

### Service Not Auto-Started
The service is enabled but **NOT** automatically started after installation because:
- Users need to configure their API key first
- Actions need to be reviewed and customized
- Prevents service from failing immediately after install

## Package Contents

When you install the RPM/DEB package, you get:

```
/usr/bin/rootly-edge-connector              # Main binary
/etc/rootly-edge-connector/
  ├── config.example.yml                    # Example config
  └── actions.example.yml                   # Example actions
/usr/lib/systemd/system/
  └── rootly-edge-connector.service         # Systemd service
/usr/share/doc/rootly-edge-connector/
  ├── README.md                              # Documentation
  └── LICENSE                                # License
/opt/rootly-edge-connector/
  ├── bin/                                   # (empty, for custom scripts)
  └── scripts/                               # (empty, for custom scripts)
/var/log/rootly-edge-connector/             # Log directory
```

## Publishing Workflow

Packages are automatically built and published when you create a git tag:

```bash
git tag v1.0.0
git push origin v1.0.0
```

GitHub Actions will:
1. Build binaries for all platforms
2. Create RPM packages (for RHEL, Fedora, CentOS, Rocky Linux, etc.)
3. Create DEB packages (for Debian, Ubuntu, etc.)
4. Attach packages to GitHub release
5. Calculate checksums

Users can then install with:

```bash
# RPM (Red Hat, Fedora, CentOS, Rocky, etc.)
sudo rpm -ivh rootly-edge-connector-1.0.0.x86_64.rpm

# DEB (Debian, Ubuntu, etc.)
sudo dpkg -i rootly-edge-connector_1.0.0_amd64.deb
```
