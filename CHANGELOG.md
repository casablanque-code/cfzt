# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.8.3] - 2026-08-06

### Fixed

- **Windows:** Task Scheduler actions no longer go through `cmd.exe /c ... >> log 2>&1`. That wrapper meant every argument (cloudflared's path, the config path, the log path) was parsed three times before the real process saw it — Task Scheduler's XML, cmd.exe's own argument parsing, then cmd.exe re-parsing its `/c` string a second time — and was the root cause of tunnels that ran fine manually but failed under Task Scheduler with `Last Result: 1`. `zt up`/`zt watchdog enable` now register the task to call `zt.exe` directly via a new hidden `zt internal run` entrypoint, which opens the log file itself and execs the real command with real argv — one quoting layer instead of three
- **Windows:** `zt up` and `zt watchdog enable` no longer report success the instant `schtasks /run` accepts the request — they now poll briefly (up to ~800ms) to confirm the task is actually still running, and fail with the tail of its log if it exited immediately. Previously a cloudflared that crashed on startup (bad config, port already bound, etc.) still looked like a fully successful `zt up`, showing as inactive only on a later `zt status`/`zt doctor`
- **Windows:** `zt restart` gets the same immediate-crash detection as `zt up`
- **Windows:** fixed a regression from the cmd.exe removal above — `zt internal run` (and therefore every tunnel/watchdog task) was popping a visible console window on the desktop, and closing it killed cloudflared along with it. zt.exe now hides its own console window on startup when running as `internal run`, and runs the child in its own process group so it no longer shares fate with zt's console
- `zt list`'s "no tunnels" hint and the root `zt --help` text were still showing bare `zt up <name> <port>` — stale since v0.8.0 made `--allow`/`--public` required. Both now show a working example
- `zt --help` now groups commands (tunnel commands / manifest commands / system) instead of one flat alphabetical list, and the root help text no longer duplicates a hand-maintained command list that had drifted out of sync with the real command set

## [0.8.2] - 2026-08-05

### Fixed

- **Windows:** Task Scheduler actions no longer go through `cmd.exe /c ... >> log 2>&1`. That wrapper meant every argument (cloudflared's path, the config path, the log path) was parsed three times before the real process saw it — Task Scheduler's XML, cmd.exe's own argument parsing, then cmd.exe re-parsing its `/c` string a second time — and was the root cause of tunnels that ran fine manually but failed under Task Scheduler with `Last Result: 1`. `zt up`/`zt watchdog enable` now register the task to call `zt.exe` directly via a new hidden `zt internal run` entrypoint, which opens the log file itself and execs the real command with real argv — one quoting layer instead of three
- **Windows:** `zt up` and `zt watchdog enable` no longer report success the instant `schtasks /run` accepts the request — they now poll briefly (up to ~800ms) to confirm the task is actually still running, and fail with the tail of its log if it exited immediately. Previously a cloudflared that crashed on startup (bad config, port already bound, etc.) still looked like a fully successful `zt up`, showing as inactive only on a later `zt status`/`zt doctor`
- **Windows:** `zt restart` gets the same immediate-crash detection as `zt up`
- **Windows:** the previous fix introduced a regression — `zt internal run` (and therefore every tunnel/watchdog task) popped a visible console window on the desktop, and closing it killed cloudflared along with it. zt.exe now hides its own console window on startup when running as `internal run`, and runs the child in its own process group so it no longer shares fate with zt's console
- `zt list`'s "no tunnels" hint and the root `zt --help` text were still showing bare `zt up <name> <port>` — stale since v0.8.0 made `--allow`/`--public` required. Both now show a working example
- `zt --help` now groups commands (tunnel commands / manifest commands / system) instead of one flat alphabetical list, and the root help text no longer duplicates a hand-maintained command list that had drifted out of sync with the real command set

## [0.8.1] - 2026-08-05

### Fixed

- `zt doctor` and `zt restart` recovery hints ("run: zt down X && zt up X ...") now reconstruct the tunnel's actual `--allow`/`--public`/`--tcp` flags instead of printing a bare `zt up <name> <port>` — which failed outright after v0.8.0 made an access-policy flag required
- **Windows:** `zt up`'s scheduled task now actually starts immediately instead of only being registered to start at the next logon — previously `zt up` reported success (DNS, ingress, service all created) while cloudflared never actually launched, showing as an inactive tunnel in `zt status`/`zt doctor` right after a clean `zt up`
- Shell tab-completion for `zt down`/`status`/`restart`/`logs` was silently broken — argument reordering meant to support `zt <name> down` was corrupting cobra's internal completion protocol

### Added

- `zt list` and `zt status` now show each tunnel's protocol and access policy (public vs. Zero Trust with email count)

## [0.8.0] - 2026-08-04

### Changed

- **Breaking:** `zt up` (and `zt apply`) now requires either `--allow <email>` or `--public` — omitting both used to still create a Zero Trust Access application, but with an empty `--allow` list it got a bypass policy with an `Everyone` include rule, which disables Access enforcement entirely. In practice `zt up grafana 3000` and `zt up grafana 3000 --public` granted the same unauthenticated public access, contradicting the "Zero Trust" pitch. Any script or CI job calling `zt up` without an access flag needs one added
- `zt up`'s DNS upsert no longer deletes *any* pre-existing record with the target hostname — only a record that looks like zt's own (a CNAME to `*.cfargotunnel.com`, e.g. a stale tunnel from a previous run) is replaced automatically. A real A/AAAA record or a CNAME to something else is left alone and `zt up` refuses with an explanation; pass the new `--force` flag (also available as `force:` in `zt.yaml` manifests) to replace it anyway
- `--public` now prints a visible warning naming the hostname instead of a plain checkmark, so it's harder to miss scrolling past — chosen over a blocking confirmation prompt specifically so scripted/CI use (e.g. PR preview environments) isn't interrupted

### Added

- `zt version` (and `zt --version`) now checks GitHub for a newer release and prints an upgrade hint if one's available; `zt doctor` surfaces the same check as an informational line. Best-effort and silent on any failure (offline, GitHub unreachable) — opt out entirely with `ZT_NO_UPDATE_CHECK=1`

### Fixed

- `doctor` no longer claims "systemd linger enabled" on macOS/Windows — that check is hardcoded to always pass there (LaunchAgents and logon-trigger scheduled tasks already auto-start, there's no separate linger toggle), so the message was simply false on those platforms. Now reports what's actually being confirmed ("systemd"/"launchd"/"scheduled task" configured to auto-start at login), via the same `service.ManagerName()` used elsewhere
- `restart` printed a hardcoded `systemctl --user restart zt-<name>` on every platform, including Windows — missed when the same class of bug was fixed elsewhere. Now uses `service.ManagerName()` like `up`/`down`/`doctor`/`ls` already do

### Added

- Homebrew tap (self-hosted in this repo's `Formula/` folder, macOS + Linux) — `brew tap casablanque-code/cfzt https://github.com/casablanque-code/cfzt && brew install zt`. Auto-updates on every release via a new release.yml step that regenerates the whole formula (not a partial field patch — the Scoop bucket already broke once from exactly that)

### Internal

- Significantly expanded test coverage across `cmd/zt` and `internal/cloudflared` (previously thin or untested: config generation, version parsing, and most CLI command error paths). Caught one real bug along the way: `cloudflared.Version.TooOld()` reported `true` for an unparseable `--version` output instead of the intended "don't block on what we don't recognize" — fixed
- Cloudflare API client is now injected via a swappable `newCFClient` var instead of called directly, enabling the above test coverage without hitting the real API

## [0.7.2] - 2026-08-02

### Fixed

- Windows: schtasks error output is emitted in the console's active codepage (e.g. CP866 on ru-RU systems) and was being captured/printed assuming UTF-8, turning every schtasks failure into mojibake. Console is now switched to UTF-8 (65001) once per process before any schtasks call
- Windows: `Install()`/`Uninstall()`/`InstallWatchdog()`/`UninstallWatchdog()` now explain the most common real-world cause of "Access Denied" — a `zt-*` task previously created or touched from an elevated (Administrator) prompt can't be overwritten from a normal session, even with `/f` — and give the exact command to clear it
- Windows: `IsActive`/`WatchdogIsActive` parsed schtasks' localized `Status:`/`Running` text output, which silently always returned `false` on non-English Windows (e.g. ru-RU). Now uses PowerShell's `Get-ScheduledTask`, whose `State` is a culture-invariant enum

## [0.7.1] - 2026-08-02

### Added

- Windows: Scoop bucket (self-hosted in this repo's `bucket/` folder) — `scoop bucket add cfzt https://github.com/casablanque-code/cfzt && scoop install cfzt/zt`

### Fixed

- Windows: `zt up` and `zt watchdog enable` failed to install a real service ("ERROR: The task XML is malformed") whenever the generated command contained an XML special character — notably the `&` in the cmd.exe log-redirection wrapper's `2>&1`, which broke on every install. Silently fell back to PID mode (no auto-restart) instead of erroring loudly. Task XML values are now properly XML-escaped
- CI: Scoop bucket auto-update never actually ran — `release: types: [published]` doesn't fire for releases created with the default `GITHUB_TOKEN`, so `scoop update zt` kept reporting 0.7.0 as latest after this release shipped. Bucket bump now happens inline in the release workflow itself instead of a separate triggered workflow

## [0.7.0] - 2026-08-01

### Added

- Windows: `zt up` now installs a real Task Scheduler service (`zt-<name>`, logon trigger, restart-on-failure) instead of only running cloudflared as a directly tracked process — gives Windows the same auto-start/auto-restart guarantee systemd/launchd provide on Linux/macOS, with no admin rights or stored credentials required
- Windows: `zt watchdog` is now available, running as its own `zt-watchdog` scheduled task
- CI: `windows-latest` added to the build/vet/test matrix (previously windows-tagged files were only cross-compiled at release time, with no tests run)

### Fixed

- `zt up`/`down`/`doctor`/`ls` no longer hardcode "systemd"/`.service` in their output — these strings were unreachable on Windows before this release (`IsInstalled` always returned `false` there) but would have been actively misleading once Task Scheduler support landed

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
