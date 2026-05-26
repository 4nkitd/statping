# Changelog

All notable changes to this project are documented here.

Format: [Keep a Changelog](https://keepachangelog.com/en/1.1.0/). Versioning: [SemVer](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- `CHANGELOG.md` and `CONTRIBUTING.md`
- README disambiguation note clarifying this project is unrelated to `statping/statping`

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

[Unreleased]: https://github.com/4nkitd/statping/compare/v1.1.2...HEAD
[v1.1.2]: https://github.com/4nkitd/statping/compare/v1.0.0...v1.1.2
[v1.0.0]: https://github.com/4nkitd/statping/compare/v0.1.0...v1.0.0
[v0.1.0]: https://github.com/4nkitd/statping/releases/tag/v0.1.0
