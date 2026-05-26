# Contributing to statping

Local-first website monitoring with a Bubble Tea TUI, macOS menu-bar tray, and native notifications.

## Dev setup

Prereqs:

- Go 1.21+
- macOS for the full experience (tray + notifications). Linux/Windows builds work but tray/notifications may be degraded.
- CGO enabled (SQLite + systray) — install Xcode command-line tools (`xcode-select --install`) on macOS, or `build-essential` on Debian/Ubuntu.

```bash
git clone https://github.com/4nkitd/statping.git
cd statping
go build -o statping ./cmd/statping
./statping start
```

## Running tests

```bash
go test ./...
```

(Test coverage is light — adding tests with new code is appreciated.)

## Project layout

```
statping/
├── cmd/statping/         # main entry point
├── internal/
│   ├── app/              # Bubble Tea TUI
│   ├── checker/          # HTTP probe logic
│   ├── store/            # SQLite storage layer (gorm-based)
│   ├── notifier/         # macOS notifications
│   ├── tray/             # system tray (getlantern/systray)
│   └── ...
├── Formula/              # Homebrew formula
├── build.sh
└── README.md
```

## Adding a feature

- **New check type** (TCP, ping, DNS, cert expiry): add to `internal/checker/`, gate behind a `Type` field on the monitor model.
- **New notifier** (webhook, Slack, ntfy): implement the `Notifier` interface in `internal/notifier/`, wire into the dispatcher.
- **TUI change**: edit `internal/app/`, update README keybindings table if user-facing.

## Branches and commits

- Branch from `master`: `feat/<name>`, `fix/<name>`, `docs/<name>`.
- Conventional Commits encouraged (see commit history for the style we use).

## PR checklist

- [ ] `go build ./cmd/statping` succeeds
- [ ] `go vet ./...` clean
- [ ] `go test ./...` passes
- [ ] Manually exercised in a terminal (state which: iTerm2, kitty, Alacritty, etc.)
- [ ] README updated if behavior changed
- [ ] `CHANGELOG.md` entry under `## [Unreleased]`
- [ ] If touching the Homebrew formula, ensure the SHA256 + version are updated

## Releases

Maintainers only. Tag `vX.Y.Z`, then run goreleaser locally (Actions disabled at the account level for now):

```bash
git tag vX.Y.Z
git push origin vX.Y.Z
GITHUB_TOKEN=<token> goreleaser release --clean
# bump Formula/statping.rb to match new SHAs
```

## Reporting issues

[GitHub issues](https://github.com/4nkitd/statping/issues).
