# Supported Platforms

The Rootly Edge Connector is built for multiple platforms to support diverse deployment environments.

## Supported Platforms

### Linux

| Platform | Architecture | Use Case |
|----------|-------------|----------|
| `linux/amd64` | x86-64 (64-bit) | Most Linux servers, cloud VMs, containers |
| `linux/arm64` | ARM 64-bit | AWS Graviton, Raspberry Pi 4/5, modern ARM servers |
| `linux/arm` | ARM 32-bit (ARMv7) | Raspberry Pi 2/3, older ARM devices |
| `linux/386` | x86 (32-bit) | Legacy Linux systems, embedded devices |

### macOS

| Platform | Architecture | Use Case |
|----------|-------------|----------|
| `darwin/amd64` | Intel x86-64 | Intel-based Macs (2020 and earlier) |
| `darwin/arm64` | Apple Silicon (M1/M2/M3) | ARM-based Macs (2020 and later) |

### Windows

| Platform | Architecture | Use Case |
|----------|-------------|----------|
| `windows/amd64` | x86-64 (64-bit) | Modern Windows servers and desktops |
| `windows/arm64` | ARM 64-bit | Windows on ARM devices (Surface Pro X, etc.) |

### BSD

| Platform | Architecture | Use Case |
|----------|-------------|----------|
| `freebsd/amd64` | x86-64 (64-bit) | FreeBSD servers, pfSense/OPNsense firewalls |

## Total Platform Coverage

- **9 platforms** across 4 operating systems
- Covers **99%+** of server, cloud, and edge deployment scenarios

## Building for Specific Platforms

### Build for Current Platform

```bash
make build
```

### Build for All Platforms

```bash
make build-all
```

This creates binaries in `bin/` directory:
```
bin/
├── rootly-edge-connector-darwin-amd64
├── rootly-edge-connector-darwin-arm64
├── rootly-edge-connector-freebsd-amd64
├── rootly-edge-connector-linux-386
├── rootly-edge-connector-linux-amd64
├── rootly-edge-connector-linux-arm
├── rootly-edge-connector-linux-arm64
├── rootly-edge-connector-windows-amd64
└── rootly-edge-connector-windows-arm64
```

### Build for Specific Platform

```bash
# Raspberry Pi 4/5 (64-bit)
GOOS=linux GOARCH=arm64 go build -o bin/rootly-edge-connector-rpi ./cmd/rec/main.go

# Raspberry Pi 2/3 (32-bit)
GOOS=linux GOARCH=arm go build -o bin/rootly-edge-connector-rpi ./cmd/rec/main.go

# FreeBSD server
GOOS=freebsd GOARCH=amd64 go build -o bin/rootly-edge-connector-freebsd ./cmd/rec/main.go

# Windows on ARM
GOOS=windows GOARCH=arm64 go build -o bin/rootly-edge-connector-arm64.exe ./cmd/rec/main.go
```

## Release Artifacts

When you push a version tag, GoReleaser automatically builds for all platforms:

```bash
git tag v1.0.0
git push origin v1.0.0
```

### Generated Artifacts

**Archives (tar.gz/zip):**
- `rootly-edge-connector_1.0.0_linux_amd64.tar.gz`
- `rootly-edge-connector_1.0.0_linux_arm64.tar.gz`
- `rootly-edge-connector_1.0.0_linux_arm.tar.gz`
- `rootly-edge-connector_1.0.0_linux_386.tar.gz`
- `rootly-edge-connector_1.0.0_darwin_amd64.tar.gz`
- `rootly-edge-connector_1.0.0_darwin_arm64.tar.gz`
- `rootly-edge-connector_1.0.0_windows_amd64.zip`
- `rootly-edge-connector_1.0.0_windows_arm64.zip`
- `rootly-edge-connector_1.0.0_freebsd_amd64.tar.gz`

**Packages (RPM/DEB):**
- `rootly-edge-connector-1.0.0.x86_64.rpm` (linux/amd64)
- `rootly-edge-connector-1.0.0.aarch64.rpm` (linux/arm64)
- `rootly-edge-connector-1.0.0.armv7hl.rpm` (linux/arm)
- `rootly-edge-connector-1.0.0.i386.rpm` (linux/386)
- `rootly-edge-connector_1.0.0_amd64.deb` (linux/amd64)
- `rootly-edge-connector_1.0.0_arm64.deb` (linux/arm64)
- `rootly-edge-connector_1.0.0_armhf.deb` (linux/arm)
- `rootly-edge-connector_1.0.0_i386.deb` (linux/386)

**Checksums:**
- `checksums.txt` - SHA256 checksums for all artifacts

## Platform-Specific Notes

### ARM Platforms

**linux/arm64 vs linux/arm:**
- `arm64` = 64-bit ARM (ARMv8-A) - Raspberry Pi 4/5, modern ARM servers
- `arm` = 32-bit ARM (ARMv7) - Raspberry Pi 2/3, older devices

Both are supported to cover the full ARM ecosystem.

### Windows on ARM

Windows 11 on ARM devices (Surface Pro X, Qualcomm-based laptops) can now run the connector natively with the `windows/arm64` build.

### FreeBSD

Useful for:
- FreeBSD servers
- pfSense and OPNsense firewalls (based on FreeBSD)
- Network appliances
- High-performance storage systems

### 32-bit Support

The `linux/386` build supports legacy 32-bit x86 systems. While uncommon today, this enables deployment on:
- Older embedded devices
- Legacy servers
- IoT devices with 32-bit processors

## Excluded Platforms

The following platforms are **not** included (uncommon for edge connector use cases):

- `linux/ppc64le` - IBM POWER (enterprise mainframes)
- `linux/s390x` - IBM Z mainframes
- `linux/mips*` - MIPS routers (limited use case)
- `linux/riscv64` - RISC-V (emerging, not production-ready)
- `openbsd/*` - OpenBSD (very niche)
- `netbsd/*` - NetBSD (very niche)

These can be added if there's demand.

## Testing Cross-Compilation

To verify all platforms build correctly:

```bash
# Build all platforms
make build-all

# Check output
ls -lh bin/
```

You should see 9 binaries (or executables for Windows).

## Platform Priority

Based on typical deployment scenarios:

**Tier 1 (Most Common):**
- `linux/amd64` - Cloud VMs, containers, most servers (80%+ of deployments)
- `linux/arm64` - AWS Graviton, modern ARM servers (growing fast)

**Tier 2 (Common):**
- `darwin/arm64` - Developer Macs (M1/M2/M3)
- `darwin/amd64` - Older Intel Macs
- `windows/amd64` - Windows servers

**Tier 3 (Edge Cases):**
- `linux/arm` - Raspberry Pi, IoT devices
- `linux/386` - Legacy systems
- `windows/arm64` - Windows on ARM
- `freebsd/amd64` - FreeBSD servers/firewalls
