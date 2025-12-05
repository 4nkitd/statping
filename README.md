# Statping - Website Monitoring CLI

A beautiful terminal-based website monitoring tool with TUI interface, system tray support, and macOS notifications.

## Features

- 📊 **Beautiful TUI** - Interactive terminal interface using Bubble Tea
- 🔔 **Native Notifications** - macOS alerts when sites go down/recover
- 🖥️ **System Tray** - Persistent menu bar monitoring with colored status icons
- 🚀 **Auto-Start** - Launch automatically on login via LaunchAgent
- 💾 **SQLite Storage** - Persistent storage at `~/.config/statping/statping.db`
- ✅ **Status Code Checks** - Verify expected HTTP status codes
- 🔍 **Keyword Matching** - Case-insensitive regex search in responses
- ⏱️ **Configurable Intervals** - Per-monitor check intervals
- 📈 **Real-Time Dashboard** - Live graphs with response time sparklines
- 🚨 **Incident Tracking** - Downtime history with duration

## Installation

```bash
# Build from source
go build -o statping ./cmd/statping

# Move to PATH
cp statping ~/bin/
# or
sudo mv statping /usr/local/bin/
```

## Quick Start

```bash
# Add a monitor
statping add https://example.com -n "Example Site"

# Start system tray (recommended)
statping tray

# Enable auto-start on login
statping enable
```

## Usage

### System Tray (Recommended)
```bash
statping tray
```
Runs persistent monitoring in your macOS menu bar with colored status icons:
- 🟢 **Green** = All monitors operational
- 🟡 **Yellow** = Some monitors slow (>1s response)
- 🔴 **Red** = One or more monitors down

Click the icon to see individual monitor status and response times.

### Auto-Start on Login
```bash
# Enable auto-start (creates macOS LaunchAgent)
statping enable

# Check auto-start status
statping status

# Disable auto-start
statping disable
```

### Interactive TUI
```bash
statping start
```

### Real-Time Dashboard with Graphs
```bash
statping dashboard
```
The dashboard shows:
- 📊 **Sparkline graphs** of response times (last 60 checks)
- 📈 **Live metrics**: Uptime %, Avg/Min/Max response times
- 🔴🟢 **Status indicators**: Color-coded for up/down/unknown
- 📋 **Summary cards**: Quick overview of all monitor statuses

### Daemon Mode (Headless)
```bash
statping daemon
```

### CLI Commands

```bash
# Add a monitor
statping add https://example.com --name "Example Site"

# Add with all options
statping add https://api.example.com \
  --name "API Server" \
  --interval 30 \
  --timeout 5 \
  --codes "200,201" \
  --keywords "success,ok"

# List all monitors
statping list

# Remove a monitor
statping remove <id>
```

### Command Reference

| Command | Description |
|---------|-------------|
| `start` | Start TUI with monitoring |
| `dashboard` | Real-time dashboard with graphs |
| `tray` | Run in system tray (menu bar) |
| `daemon` | Run headless in background |
| `add <url>` | Add a new monitor |
| `list` | List all monitors |
| `remove <id>` | Remove a monitor |
| `enable` | Enable auto-start on login |
| `disable` | Disable auto-start |
| `status` | Check auto-start status |

## TUI Keybindings

| Key | Action |
|-----|--------|
| `a` | Add new monitor |
| `e` | Edit selected monitor |
| `d` | Delete selected monitor |
| `t` | Toggle enable/disable |
| `Enter` | View details |
| `r` | Refresh |
| `q` | Quit / Back |
| `j/k` or `↑/↓` | Navigate |
| `Tab` | Next field (in forms) |
| `Esc` | Cancel / Back |

## Configuration

Each monitor can be configured with:

- **Name** - Display name for the monitor
- **URL** - The URL to check
- **Check Interval** - How often to check (seconds, default: 60)
- **Timeout** - Request timeout (seconds, default: 10)
- **Expected Codes** - Comma-separated status codes (default: 200)
- **Keywords** - Comma-separated keywords to find in response (optional)

## Notifications

- 🔴 **Down Alert** - After 3 consecutive failures
- ✅ **Recovery Alert** - When site comes back up
- ⏰ **Cooldown** - 5 minutes between repeat alerts

## Data Storage

All data is stored in SQLite at:
```
~/.config/statping/statping.db
```

Logs (when running via LaunchAgent):
```
~/.config/statping/statping.log
~/.config/statping/statping.err
```

## Requirements

- macOS (for system tray and notifications)
- Go 1.21+ (for building)

## License

MIT
