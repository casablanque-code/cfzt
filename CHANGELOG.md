# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.4.0] - 2026-06-20

### Added
- `zt watchdog enable` / `disable` / `status` — background service that automatically recovers tunnels stuck on the HTTP/2 fallback after a transient UDP/QUIC blip (cloudflared never retries QUIC on its own — see [cloudflare/cloudflared#1534](https://github.com/cloudflare/cloudflared/issues/1534))
- `internal/watchdog` package — incremental log tailing, exponential backoff per tunnel (10min → 60min cap), separate runtime state file (`~/.zt-watchdog-state.json`) to avoid contending with interactive commands
- `cloudflared.LogPath()` helper
- `zt doctor` now reports watchdog status
- 14 tests covering incremental log scanning, backoff math, and runtime state corruption recovery

### Changed
- `restartTunnel(name)` extracted from `zt restart` so the watchdog reuses identical restart logic
- Only tunnels with `protocol: auto` (the default) are watched — tunnels pinned via `--protocol`/`--tcp` are left untouched, as that's a deliberate user choice

## [0.3.0] - 2026-06-19

### Added
- `zt export` — snapshot all managed tunnels to a portable `zt.yaml` manifest (credentials and tunnel IDs excluded, safe to commit to git)
- `zt apply <file>` — apply a `zt.yaml` manifest on any machine; diffs against local state and creates only what is missing, never deletes automatically
- `internal/manifest` package with `Manifest`/`ServiceSpec` types, `Load` (with validation) and `Save`
- State now persists tunnel intent: `Public`, `AllowEmails`, `DockerDetect` fields added to `Tunnel` — backward compatible with existing state files
- Tests for `internal/manifest` (roundtrip, validation, edge cases) and `internal/state` (new fields, backward compat, omitempty)
- `gopkg.in/yaml.v3` as direct dependency

### Fixed
- Missing `//go:build !windows` tag in `runner.go` caused redeclared symbol errors when cross-compiling in CI
- `go.sum` now contains correct `golang.org/x/sys` transitive checksums — `go build` works without `go mod tidy`

### Changed
- `createTunnel(tunnelOpts)` extracted from `runUp` so `apply` reuses the exact same creation logic without duplication

## [0.2.2] - 2026-06-12

### Added
- `zt doctor`: check linger status for systemd user services (warns if linger is disabled — tunnels won't survive logout)
- Tunnel status check endpoint in `zt status`
- CI badge in README

### Fixed
- Linter issues; golangci-lint config inlined into workflow

## [0.2.1] - 2026-06-06

### Added
- SHA-256 checksum verification in `install.sh`
- `zt doctor` — health check utility: validates cloudflared installation, API token, DNS resolution, tunnel and service status
- API token validation on `zt init`

## [0.2.0] - 2026-05-28

### Added
- Tunnels now run as systemd user services (Linux) / LaunchAgents (macOS) — survive reboots automatically
- `--tcp` flag to force HTTP/2 protocol (for ISPs blocking QUIC/UDP)
- `--protocol` flag: `auto` (default), `quic`, `http2`
- `zt logs` command with `-n <lines>` and `-f` (follow) flags, colorized output
- `--docker` flag — auto-detect port from a running Docker container by name
- cloudflared minimum version check (2023.x) on `zt up`

## [0.1.3] - 2026-05-26

### Fixed
- Build tags for Unix/Windows syscall compatibility (`runner.go` / `runner_windows.go`)

[0.3.0]: https://github.com/casablanque-code/cfzt/compare/v0.2.2...v0.3.0
[0.2.2]: https://github.com/casablanque-code/cfzt/compare/v0.2.1...v0.2.2
[0.2.1]: https://github.com/casablanque-code/cfzt/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/casablanque-code/cfzt/compare/v0.1.3...v0.2.0
[0.1.3]: https://github.com/casablanque-code/cfzt/compare/v0.1.2...v0.1.3
