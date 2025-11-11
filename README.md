# Cloudflare Dynamic DNS Updater

A lightweight, cross-platform daemon that automatically updates Cloudflare DNS records when your public IP address changes. Perfect for home servers, self-hosted services, and dynamic IP environments.

## Features

- **Dual-Stack Support**: Updates both IPv4 (A) and IPv6 (AAAA) records
- **Multi-Record Management**: Update multiple DNS records from a single configuration
- **Daemon Mode**: Runs continuously in the background, checking for IP changes
- **Smart Updates**: Only calls the Cloudflare API when your IP actually changes
- **Cross-Platform**: Works on Linux, macOS, and Windows
- **Auto-Install Service**: Built-in commands to install as a system service
  - Linux: systemd service
  - macOS: launchd service
  - Windows: Task Scheduler
- **Graceful Shutdown**: Properly handles SIGTERM/SIGINT signals
- **Reliable IP Detection**: Multiple fallback services for IP detection
- **Easy Configuration**: Simple YAML configuration file

## Installation

### Download Pre-built Binaries

Download the latest release for your platform from the [Releases](https://github.com/MrLonely14/cf-ddns/releases) page.

**Linux:**
```bash
# Download and extract (replace VERSION with actual version, e.g., v1.0.0)
wget https://github.com/MrLonely14/cf-ddns/releases/download/VERSION/cf-ddns_VERSION_linux_amd64.tar.gz
tar -xzf cf-ddns_VERSION_linux_amd64.tar.gz
sudo mv cf-ddns /usr/local/bin/
```

**macOS:**
```bash
# Intel Macs
wget https://github.com/MrLonely14/cf-ddns/releases/download/VERSION/cf-ddns_VERSION_darwin_amd64.tar.gz
tar -xzf cf-ddns_VERSION_darwin_amd64.tar.gz
sudo mv cf-ddns /usr/local/bin/

# Apple Silicon (M1/M2/M3)
wget https://github.com/MrLonely14/cf-ddns/releases/download/VERSION/cf-ddns_VERSION_darwin_arm64.tar.gz
tar -xzf cf-ddns_VERSION_darwin_arm64.tar.gz
sudo mv cf-ddns /usr/local/bin/
```

**Windows:**
1. Download `cf-ddns_VERSION_windows_amd64.zip` from releases
2. Extract the ZIP file
3. Run `cf-ddns.exe` from Command Prompt or PowerShell

### Build from Source

Requirements: Go 1.21 or later

```bash
git clone https://github.com/MrLonely14/cf-ddns.git
cd cf-ddns
go build -o cf-ddns
```

## Quick Start

### 1. Get Your Cloudflare API Token

1. Go to [Cloudflare Dashboard](https://dash.cloudflare.com/profile/api-tokens)
2. Click "Create Token"
3. Use the "Edit zone DNS" template
4. Select your zone(s)
5. Copy the generated token

### 2. Find Your Zone ID

1. Go to your domain in Cloudflare Dashboard
2. Scroll down on the Overview page
3. Find "Zone ID" in the right sidebar
4. Copy the Zone ID

### 3. Create Configuration File

Copy the example configuration:
```bash
cp config/config.example.yaml config.yaml
```

Edit `config.yaml`:
```yaml
cloudflare:
  api_token: "your-cloudflare-api-token"

check_interval: "5m"

records:
  - zone_id: "your-zone-id"
    name: "home.example.com"
    types: ["A", "AAAA"]
    ttl: 120
    proxied: false
```

### 4. Run the Daemon

```bash
# Run in foreground (for testing)
./cf-ddns run -config config.yaml

# Or install as a system service
sudo ./cf-ddns install -config /etc/cf-ddns/config.yaml
```

## Usage

### Commands

```bash
cf-ddns run [flags]          # Run the daemon (default)
cf-ddns install [flags]      # Install as system service
cf-ddns uninstall            # Uninstall system service
cf-ddns status               # Check service status
cf-ddns version              # Show version
cf-ddns help                 # Show help message
```

### Flags

#### Run Command
- `-config string` - Path to configuration file (default: `config.yaml`)

#### Install Command
- `-config string` - Path to configuration file (default: `/etc/cf-ddns/config.yaml`)
- `-user string` - User to run the service as (default: current user)

## Configuration

### Example Configuration

```yaml
cloudflare:
  # Your Cloudflare API token with DNS edit permissions
  api_token: "your-cloudflare-api-token-here"

# How often to check for IP address changes
check_interval: "5m"  # Valid units: s, m, h

# DNS records to update
records:
  # Home server with IPv4 and IPv6
  - zone_id: "abc123..."
    name: "home.example.com"
    types: ["A", "AAAA"]
    ttl: 120
    proxied: false

  # VPN server (IPv4 only)
  - zone_id: "abc123..."
    name: "vpn.example.com"
    types: ["A"]
    ttl: 300
    proxied: false

  # Root domain with Cloudflare proxy
  - zone_id: "abc123..."
    name: "example.com"
    types: ["A"]
    ttl: 120
    proxied: true
```

### Configuration Options

- **cloudflare.api_token** (required): Cloudflare API token with DNS edit permissions
- **check_interval** (required): How often to check for IP changes (e.g., `5m`, `10m`, `1h`)
- **records** (required): List of DNS records to manage

#### Record Options

- **zone_id** (required): Cloudflare Zone ID
- **name** (required): Full DNS record name (e.g., `home.example.com`)
- **types** (required): List of record types to update (`A` for IPv4, `AAAA` for IPv6)
- **ttl** (required): Time to live in seconds (60-86400)
- **proxied** (required): Whether to proxy through Cloudflare (true/false)

## Installing as a Service

### Linux (systemd)

```bash
# Install the service (this creates config.example.yaml automatically)
sudo ./cf-ddns install -config /etc/cf-ddns/config.yaml

# Copy and edit the example config
sudo cp /etc/cf-ddns/config.example.yaml /etc/cf-ddns/config.yaml
sudo nano /etc/cf-ddns/config.yaml
# (Edit with your Cloudflare API token and zone details)

# Start the service
sudo systemctl start cf-ddns
sudo systemctl enable cf-ddns

# Check status
sudo systemctl status cf-ddns

# View logs
sudo journalctl -u cf-ddns -f

# Restart after config changes
sudo systemctl restart cf-ddns

# Uninstall
sudo ./cf-ddns uninstall
```

### macOS (launchd)

```bash
# Install the service (this creates config.example.yaml automatically)
./cf-ddns install -config ~/.config/cf-ddns/config.yaml

# Copy and edit the example config
cp ~/.config/cf-ddns/config.example.yaml ~/.config/cf-ddns/config.yaml
nano ~/.config/cf-ddns/config.yaml
# (Edit with your Cloudflare API token and zone details)

# Service starts automatically after installation

# View logs
tail -f /tmp/cf-ddns.log

# Restart after config changes
launchctl unload ~/Library/LaunchAgents/com.cf-ddns.plist
launchctl load ~/Library/LaunchAgents/com.cf-ddns.plist

# Uninstall
./cf-ddns uninstall
```

### Windows (Task Scheduler)

```powershell
# Run PowerShell as Administrator

# Install the service (this creates config.example.yaml automatically)
.\cf-ddns.exe install -config "$env:ProgramData\cf-ddns\config.yaml"

# Copy and edit the example config
Copy-Item "$env:ProgramData\cf-ddns\config.example.yaml" "$env:ProgramData\cf-ddns\config.yaml"
notepad "$env:ProgramData\cf-ddns\config.yaml"
# (Edit with your Cloudflare API token and zone details)

# Check task in Task Scheduler
taskschd.msc

# Restart after config changes
Stop-ScheduledTask -TaskName "CloudflareDDNS"
Start-ScheduledTask -TaskName "CloudflareDDNS"

# Uninstall
.\cf-ddns.exe uninstall
```

## How It Works

1. **IP Detection**: The daemon detects your current public IPv4 and IPv6 addresses using multiple reliable services:
   - IPv4: ipify.org, icanhazip.com, ifconfig.me, checkip.amazonaws.com
   - IPv6: api64.ipify.org, ipv6.icanhazip.com, v6.ident.me

2. **Change Detection**: Compares current IPs with the last known IPs for each record

3. **DNS Update**: If an IP has changed, updates the corresponding Cloudflare DNS record via API

4. **Repeat**: Waits for the configured interval and checks again

## Configuration Changes

**Important**: Configuration changes require a service restart to take effect.

- **Linux**: `sudo systemctl restart cf-ddns`
- **macOS**:
  ```bash
  launchctl unload ~/Library/LaunchAgents/com.cf-ddns.plist
  launchctl load ~/Library/LaunchAgents/com.cf-ddns.plist
  ```
- **Windows**:
  ```powershell
  Stop-ScheduledTask -TaskName "CloudflareDDNS"
  Start-ScheduledTask -TaskName "CloudflareDDNS"
  ```
- **Manual/Foreground**: Press `Ctrl+C` and restart the command

## Troubleshooting

### Service Won't Start

Check the logs:
- **Linux**: `sudo journalctl -u cf-ddns -f`
- **macOS**: `tail -f /tmp/cf-ddns.log`
- **Windows**: Check Task Scheduler history

### API Token Issues

Ensure your API token has:
- Zone.DNS.Edit permissions
- Access to the correct zone(s)

Test your token:
```bash
curl -X GET "https://api.cloudflare.com/client/v4/zones" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### DNS Record Not Updating

- Verify your Zone ID is correct
- Check the record name matches exactly (including subdomain)
- Ensure the record type (A/AAAA) matches your IP version
- Check if you have IPv6 connectivity (for AAAA records)

### IPv6 Detection Fails

If you don't have IPv6 connectivity, remove `"AAAA"` from the `types` list in your config.

## Security Considerations

- **API Token**: Keep your API token secure. Never commit `config.yaml` to version control
- **File Permissions**: Ensure config file has restricted permissions:
  ```bash
  chmod 600 config.yaml
  ```
- **Principle of Least Privilege**: Create API tokens with only DNS edit permissions for specific zones

## Development

### Project Structure

```
cf-ddns/
├── main.go              # Entry point and CLI
├── config/              # Configuration loading
├── cloudflare/          # Cloudflare API client
├── ipdetect/            # IP detection logic
├── updater/             # Core update logic
├── installer/           # Service installation
├── templates/           # Service templates
└── .github/workflows/   # CI/CD
```

### Building

```bash
# Build for current platform
go build -o cf-ddns

# Cross-compile for all platforms
GOOS=linux GOARCH=amd64 go build -o cf-ddns-linux-amd64
GOOS=darwin GOARCH=arm64 go build -o cf-ddns-darwin-arm64
GOOS=windows GOARCH=amd64 go build -o cf-ddns-windows-amd64.exe
```

### Running Tests
- not exsist yet
```bash
go test -v ./...
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT License - see LICENSE file for details

## Acknowledgments

- [Cloudflare Go SDK](https://github.com/cloudflare/cloudflare-go)
- IP detection services: ipify, icanhazip, ifconfig.me, AWS checkip

## Support

If you encounter any issues or have questions:
1. Check the [Troubleshooting](#troubleshooting) section
2. Search existing [Issues](https://github.com/MrLonely14/cf-ddns/issues)
3. Create a new issue with:
   - Your operating system and version
   - cf-ddns version (`cf-ddns version`)
   - Relevant log output (redact sensitive information)
   - Steps to reproduce the issue
