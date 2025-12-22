# Albion Lens

[![Release Status](https://github.com/cantalupo555/albion-lens/actions/workflows/release.yml/badge.svg)](https://github.com/cantalupo555/albion-lens/actions/workflows/release.yml)
[![Latest Release](https://img.shields.io/github/v/release/cantalupo555/albion-lens)](https://github.com/cantalupo555/albion-lens/releases)
[![Go Version](https://img.shields.io/github/go-mod/go-version/cantalupo555/albion-lens)](https://github.com/cantalupo555/albion-lens/blob/master/go.mod)
[![Go Report Card](https://goreportcard.com/badge/github.com/cantalupo555/albion-lens)](https://goreportcard.com/report/github.com/cantalupo555/albion-lens)
[![Downloads](https://img.shields.io/github/downloads/cantalupo555/albion-lens/total)](https://github.com/cantalupo555/albion-lens/releases)
[![License](https://img.shields.io/badge/License-GPLv3-blue.svg)](LICENSE)

Cross-platform network analyzer for Albion Online written in Go.

## Features

- UDP packet capture for Albion Online (ports 5055/5056)
- Photon Engine protocol parser
- Protocol16 decoding
- Fame (XP) tracking with session counting
- Silver tracking
- Loot events with item names (via ao-bin-dumps)
- Discovery Mode - Discover unknown event codes
- Cross-platform (Linux, Windows, macOS)


## Prerequisites

### Linux
```bash
sudo apt-get install libpcap-dev
```

### Windows
Install [Npcap](https://npcap.com/) with WinPcap compatibility mode.

### macOS
libpcap is already included in macOS.


## Installation

### Option 1: Download (Recommended)

Download the latest release for your platform from the [Releases](https://github.com/cantalupo555/albion-lens/releases) page.

| Platform | Architecture | File |
|----------|--------------|------|
| **Linux** | x64 (Intel/AMD) | `.deb`, `.rpm`, or binary |
| **Linux** | ARM64 | `.deb`, `.rpm`, or binary |
| **Windows** | x64 (Intel/AMD) | `.exe` |
| **macOS** | Intel | `.zip` |
| **macOS** | Apple Silicon (M1/M2/M3) | `.zip` |

#### Linux (Debian/Ubuntu)
```bash
# Download and install .deb package
sudo dpkg -i albion-lens_*_linux_amd64.deb
```

#### Linux (Fedora/RHEL)
```bash
# Download and install .rpm package
sudo rpm -i albion-lens_*_linux_amd64.rpm
```

#### Windows
1. Download the `.exe` file
2. Install [Npcap](https://npcap.com/) with WinPcap compatibility mode
3. Run directly or add to PATH

#### macOS
```bash
# Extract and run
unzip albion-lens_*_macOS_arm64.zip
sudo ./albion-lens
```

### Option 2: Build from Source

Requires **Go 1.25+** and **libpcap-dev** installed.

```bash
# Clone the repository
git clone https://github.com/cantalupo555/albion-lens
cd albion-lens

# Download dependencies
go mod download

# Build
go build -o bin/albion-lens ./cmd/sniffer

# Run (requires sudo for packet capture)
sudo ./bin/albion-lens
```


## Usage

```bash
# Run (capture on all interfaces)
sudo ./albion-lens

# List network devices
sudo ./albion-lens -list

# Capture on specific device
sudo ./albion-lens -device eth0

# Debug mode (shows all packets)
sudo ./albion-lens -debug

# Discovery mode - discover new event codes
sudo ./albion-lens -discovery

# With item name resolution (ao-bin-dumps)
sudo ./albion-lens -items ../ao-bin-dumps

# Save discovered events to specific file
sudo ./albion-lens -discovery -save-discovery output/events.json

# Full combination
sudo ./albion-lens -discovery -items ../ao-bin-dumps -debug
```


### Discovery Mode

Discovery mode is essential for identifying new event codes when the game is updated:

```bash
sudo ./albion-lens -discovery
```

When enabled:
- Logs all unknown events with their parameters
- Shows a summary at the end of the session (known vs unknown events)
- Auto-saves discovered events to `output/discovered_events_YYYY-MM-DD_HH-MM-SS.json`


### Item Name Resolution

To see item names instead of numeric IDs, point to the [ao-bin-dumps](https://github.com/ao-data/ao-bin-dumps) repository:

```bash
# Clone ao-bin-dumps
git clone https://github.com/ao-data/ao-bin-dumps ../ao-bin-dumps

# Run with the -items flag
sudo ./albion-lens -items ../ao-bin-dumps
```

Albion Lens also attempts to auto-detect ao-bin-dumps in common locations.


## References

- [Photon Engine](https://www.photonengine.com/) - The networking middleware used by Albion Online
- [ao-bin-dumps](https://github.com/ao-data/ao-bin-dumps) - Data extracted from the client
- [ao-loot-logger](https://github.com/matheussampaio/ao-loot-logger) - Primary inspiration for Photon parsing


## Disclaimer

This project is for educational purposes only. Use at your own risk and in compliance with Albion Online's Terms of Service.


## License

[GNU GPLv3](LICENSE)
