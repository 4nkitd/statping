# Statping - Website Monitoring CLI

A beautiful terminal-based website monitoring tool with TUI interface and macOS notifications.

## Features

- 📊 **Beautiful TUI** - Interactive terminal interface using Bubble Tea
- 🔔 **Native Notifications** - macOS alerts when sites go down/recover
- 💾 **SQLite Storage** - Persistent storage at `~/.config/statping/statping.db`
- ✅ **Status Code Checks** - Verify expected HTTP status codes
- 🔍 **Keyword Matching** - Case-insensitive regex search in responses
- ⏱️ **Configurable Intervals** - Per-monitor check intervals
- 📈 **Statistics** - Uptime percentages and response times
- 🚨 **Incident Tracking** - Downtime history with duration

## Installation

```bash
# Build from source
go build -o statping ./cmd/statping

# Move to PATH (optional)
sudo mv statping /usr/local/bin/
```

## Usage

### Start TUI + Monitoring
```bash
./statping start
```

### Real-Time Dashboard with Graphs
```bash
./statping dashboard
```
The dashboard shows:
- 📊 **Sparkline graphs** of response times (last 60 checks)
- 📈 **Live metrics**: Uptime %, Avg/Min/Max response times
- 🔴🟢 **Status indicators**: Color-coded for up/down/unknown
- 📋 **Summary cards**: Quick overview of all monitor statuses

### Run in Background (Daemon Mode)
```bash
./statping daemon
```

### CLI Commands

```bash
# Add a monitor
./statping add https://example.com --name "Example Site"

# Add with options
./statping add https://api.example.com \
  --name "API Server" \
  --interval 30 \
  --timeout 5 \
  --codes "200,201" \
  --keywords "success,ok"

# List all monitors
./statping list

# Remove a monitor
./statping remove 1
```

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

## License

MIT
