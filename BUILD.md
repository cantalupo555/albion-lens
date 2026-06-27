# Build Instructions

This document describes how to build **albion-lens** from source on Linux, Windows, and macOS.

These instructions mirror the official release pipeline in [`.github/workflows/release.yml`](.github/workflows/release.yml), so a local build reproduces the binaries published on the [Releases](https://github.com/cantalupo555/albion-lens/releases) page.

albion-lens requires [CGO](https://go.dev/blog/cgo) because it links against **libpcap** (the packet-capture library). Each platform provides libpcap differently, so the build steps vary slightly per OS.


## Prerequisites (all platforms)

- **Go 1.26+** — [go.dev/dl](https://go.dev/dl/)
- Git to clone the repository:

```bash
git clone https://github.com/cantalupo555/albion-lens
cd albion-lens
go mod download
```

- CGO must be enabled (it is by default when a C compiler and libpcap are available).


## Linux

### 1. Install libpcap development headers

Debian / Ubuntu:

```bash
sudo apt-get install -y libpcap-dev
```

Fedora / RHEL:

```bash
sudo dnf install -y libpcap-devel
```

### 2. Build

```bash
go build -o bin/albion-lens ./cmd/tui
```

### 3. Run

Packet capture requires root privileges:

```bash
sudo ./bin/albion-lens
```


## Windows

Only **amd64** is supported. WinPcap/Npcap do not publish ARM64 binaries.

### 1. Install the WinPcap SDK (WpdPack)

Download and extract the WinPcap Developer's Pack (WpdPack 4.1.2) to `C:\` so that `C:\WpdPack\Include` and `C:\WpdPack\Lib\x64` exist:

```powershell
Invoke-WebRequest -Uri "https://www.winpcap.org/install/bin/WpdPack_4_1_2.zip" -OutFile "WpdPack.zip"
Expand-Archive -Path "WpdPack.zip" -DestinationPath "C:\"
```

You also need a C compiler (CGO requirement). Git Bash + [MinGW-w64](https://www.mingw-w64.org/) is the most common setup.

### 2. Build

Run from a shell that understands the environment-variable syntax below (Git Bash, WSL, or PowerShell with appropriate syntax adjustments):

```bash
CGO_ENABLED=1 \
  CGO_CFLAGS="-IC:/WpdPack/Include" \
  CGO_LDFLAGS="-LC:/WpdPack/Lib/x64" \
  go build -o bin/albion-lens.exe ./cmd/tui
```

### 3. Runtime requirement

Capturing packets at runtime requires [Npcap](https://npcap.com/) installed **with WinPcap compatibility mode** enabled. The SDK above is only needed for building; Npcap is the runtime driver.


## macOS

### 1. Install Xcode Command Line Tools

macOS ships libpcap as part of the Xcode Command Line Tools:

```bash
xcode-select --install
```

If the tools are already installed, the command will report so — no further action is needed.

### 2. Build

```bash
go build -o bin/albion-lens ./cmd/tui
```

### 3. Run

```bash
sudo ./bin/albion-lens
```


## Troubleshooting

| Symptom | Cause / Fix |
|---|---|
| `cgo: C compiler "gcc" not found` | No C compiler installed. Linux: `sudo apt-get install build-essential`. Windows: install MinGW-w64 and ensure `gcc` is on `PATH`. macOS: run `xcode-select --install`. |
| `pcap.h: No such file or directory` | `CGO_CFLAGS` does not point to the extracted WpdPack `Include` folder, or libpcap headers are not installed on Linux/macOS. |
| Linker errors referencing `wpcap` / `Packet` | `CGO_LDFLAGS` does not point to `C:/WpdPack/Lib/x64`, or you are building for an unsupported architecture (only amd64 is supported on Windows). |
| `You don't have permission to capture` / raw-socket error | Run the binary with administrator / root privileges (`sudo` on Linux/macOS, "Run as administrator" on Windows). |
| Build works but no traffic is captured on Windows | Npcap was not installed with WinPcap compatibility mode. Reinstall Npcap and enable that option. |


## Reference

- Release pipeline: [`.github/workflows/release.yml`](.github/workflows/release.yml)
- WinPcap Developer's Pack: [winpcap.org/devel](https://www.winpcap.org/devel.htm)
- Npcap (runtime, Windows): [npcap.com](https://npcap.com/)
- Go CGO: [go.dev/blog/cgo](https://go.dev/blog/cgo)
