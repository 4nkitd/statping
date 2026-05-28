# Changelog

All notable changes to this project are documented here.

Format: [Keep a Changelog](https://keepachangelog.com/en/1.1.0/). Versioning: [SemVer](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

---

## [v1.2.0] — 2026-05-29

### Added
- **Internet-aware checking**: when the local system is offline, monitor failures are no
  longer treated as the site being down — no false incidents, notifications, or uptime
  pollution. Connectivity is probed against trusted anchors; monitoring resumes automatically.
- "No internet — monitoring paused" / "Internet restored" notifications.
- New CLI commands: `edit`, `pause`, `resume`, and `check` (one-off probe).
- Per-monitor `MaxFailures` (failures-before-down) threshold, configurable in CLI, TUI, and web UI.
- Check-result retention/pruning to prevent unbounded database growth.
- Responsive multi-column dashboard grid with scrolling for many monitors.
- Web settings UI: live status + connectivity banner, per-card uptime/latency, and edit support.
- Landing site (`site/`) with a synthesized `DESIGN.md`, deployable via Cloudflare Workers.
- Shared `internal/icons` traffic-light generator.

### Changed
- Unified the system tray onto the shared checker (per-monitor intervals, `MaxFailures`,
  and incident tracking now apply to tray-driven checks).
- Menu-bar icon is now a traffic light reflecting overall status / connectivity.

### Fixed
- List and dashboard TUIs no longer overflow the terminal with many monitors.
- Robust ID and status-code parsing; previously-ignored database errors are now logged.
- Release workflow now checksums the published `.tar.gz` assets and regenerates the
  Homebrew formula on every release, so `brew upgrade` reliably gets the latest version.

---

## [v1.1.2] — 2025-12-10

### Changed
- Homebrew formula bumped to v1.1.2 with refreshed checksum

---

## [v1.0.0] — 2025-12-10

### Added
- Homebrew formula and updated installation instructions
- Windows build support in release workflow
- Detailed monitor analytics view with graphs
- Functionality to retrieve check results since a specific time
- Enhanced detail page styling

---

## [v0.1.0] — 2025-12-06

Initial release.

### Added
- Interactive Bubble Tea TUI dashboard with response-time sparklines
- Real-time dashboard with live metrics (uptime %, avg/min/max response times)
- System tray for macOS menu bar with colored status icons (green / yellow / red)
- macOS native notifications on down + recovery
- Persistent monitoring via macOS LaunchAgent (`statping enable` / `disable`)
- SQLite storage at `~/.config/statping/statping.db`
- Daemon mode (`statping daemon`) for headless monitoring
- Per-monitor configuration: name, URL, interval, timeout, expected status codes, keywords
- Incident tracking with downtime duration
- Web-based settings UI for systray
- 3-failure / 5-minute-cooldown notification policy

[Unreleased]: https://github.com/4nkitd/statping/compare/v1.2.0...HEAD
[v1.2.0]: https://github.com/4nkitd/statping/compare/v1.1.2...v1.2.0
[v1.1.2]: https://github.com/4nkitd/statping/compare/v1.0.0...v1.1.2
[v1.0.0]: https://github.com/4nkitd/statping/compare/v0.1.0...v1.0.0
[v0.1.0]: https://github.com/4nkitd/statping/releases/tag/v0.1.0
