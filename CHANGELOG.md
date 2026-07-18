# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.6.1] - 2026-07-18

### Added

- Test coverage: `internal/cloudflare` (0% → 79.9%), `internal/docker` (0% → 86.2%), `config` (0% → 79.3%), `internal/state/store.go` (0% → 84.8%), plus targeted tests for `cmd/zt`'s pure/orchestration logic (`tunnelStatus`, `protocolLabel`, `protocolForExport`, `runExport`, `runApply`'s diff logic, `resolveApplyPort`)

### Fixed

- `zt apply`: a Docker-flagged service with an explicit `port:` in the manifest no longer fails when Docker itself is unreachable — `resolveApplyPort()` now checks the explicit override before querying Docker, not after (previously it queried Docker unconditionally and only used the override if the query happened to succeed)
- `internal/cloudflare`: `UpsertCNAME`/`UpsertAccessApp` upsert semantics (delete-existing-before-create) now covered by tests, catching any future regression in the `zt up` rollback path
- README: removed dead `[#anchor](#anchor)` link stubs left over from earlier patches; Windows install instructions consolidated into `## Install` (were previously duplicated/split from `## Windows support`)

### Changed

- `internal/cloudflare.Client` gained an overridable `baseURL` (via `NewClientForTesting`) and `internal/docker`'s package-level `httpClient` is now swappable in tests — both internal-only, no change to public API

## [0.6.0] - 2026-07-17

### Added

- Windows: core tunnel lifecycle now works — `zt up`/`down`/`restart`/`status`/`logs`/`doctor` run cloudflared in PID-tracked mode (the same fallback mode Linux/macOS use without systemd/launchd). `zt up` reports `(no auto-restart)` explicitly when running this way.
- README: "Windows support" section documenting what works and what doesn't yet (no persistent service, no `zt watchdog` on Windows)
- README: "Installing `zt` on Windows" — PowerShell installer that downloads the latest release, verifies its sha256, and adds it to user `PATH` via `[Environment]::SetEnvironmentVariable` (not `setx`, which silently truncates PATH past 1024 characters)

### Fixed

- `cloudflared --version` detection now works on Windows (`zt doctor` previously always reported `unknown` — the Windows codepath was a hardcoded stub even though the underlying logic was already platform-agnostic)
- `.gitignore`: `dist/` build artifacts and `*.patch` files no longer show up as untracked/dirty in `git status`

### Changed

- `golang.org/x/sys` moved from an indirect to a direct dependency in `go.mod` (now imported directly for Windows process management)

## [0.5.1] - 2026-07-14

### Added

- `zt doctor`: platform-aware `cloudflared` upgrade hint — detects Homebrew (macOS) or apt (Linux, via `dpkg -S`) ownership of the `cloudflared` binary and prints the matching upgrade command (`brew upgrade cloudflared` / `sudo apt update && sudo apt install --only-upgrade cloudflared`) instead of a bare downloads link
- README: new "Installing cloudflared" section — per-platform install/upgrade table (Homebrew, apt, generic binary, Windows), replacing the single generic link in Prerequisites

### Docs

- Noted that Windows support in `zt` itself is currently partial (see `internal/service` / `internal/cloudflared` `_windows.go` stubs — process lifecycle management not yet implemented)

## [0.5.0] - 2026-07-14

### Added

- `zt status <name>` now shows the live-negotiated protocol for `auto` tunnels (`auto (http2)` / `auto (quic)`) instead of a static `auto`, via `cloudflared.DetectEffectiveProtocol()` tailing `cloudflared.log` for connection registration and startup connectivity pre-check markers
- `--version` flag
- `zt <name> status` / `logs` / `restart` / `down` now work identically to `zt status` / `logs` / `restart` / `down` `<name>`
- `zt completion bash|zsh|fish|powershell` documented in README (was already available via Cobra, just undocumented)
- README: troubleshooting entry for `502 Bad Gateway` despite an apparently healthy QUIC connection (UDP MTU/fragmentation dropping real traffic while the pre-check's small test connection still passes) — fix is `--tcp`, not automatically detectable
- README: documented that there is no `zt update` — reinstalling via `install.sh`/`go install` overwrites the binary in place

### Fixed

- `--version` / `-v` were silently non-functional: the Makefile's `-ldflags -X main.version=...` targeted a `main.version` symbol that didn't exist in the code, so the linker dropped it without error
- root `--help` didn't surface `--tcp` or shell completion, even though both existed — only visible via `zt up --help` / by knowing to look

### Tests

- 4 tests for `DetectEffectiveProtocol`, including a fixture mirroring a real degraded-QUIC pre-check log
- 8 tests for the name-first argument reordering (`zt <name> status`, flag passthrough, aliases, no-ops)

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
