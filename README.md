# zt — Zero Trust tunnel manager

[![CI](https://github.com/casablanque-code/cfzt/actions/workflows/ci.yml/badge.svg)](https://github.com/casablanque-code/cfzt/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/casablanque-code/cfzt/branch/main/graph/badge.svg)](https://codecov.io/gh/casablanque-code/cfzt)
[![Go Reference](https://pkg.go.dev/badge/github.com/casablanque-code/cfzt.svg)](https://pkg.go.dev/github.com/casablanque-code/cfzt)
[![Gitleaks](https://img.shields.io/badge/protected%20by-gitleaks-blue)](https://github.com/casablanque-code/cfzt/actions/workflows/gitleaks.yml)
[![Tiny Tool Town](https://img.shields.io/badge/featured-TinyToolTown-blue)](https://www.tinytooltown.com/tools/cfzt/)
[![Release](https://img.shields.io/github/v/release/casablanque-code/cfzt)](https://github.com/casablanque-code/cfzt/releases/latest)

One command to expose a local service through Cloudflare Zero Trust.

```bash
zt up portainer --docker --allow you@example.com
# → https://portainer.yourdomain.com  (ZT-protected, live in ~15s)
```
![zt demo](demo.gif)

## What it does

`zt up <name> <port>` automatically:

1. Creates a Cloudflare Tunnel
2. Configures ingress rules
3. Upserts a CNAME DNS record (replaces any conflicting record)
4. Creates a Zero Trust Access application with an access policy
5. Installs and starts a systemd (Linux) or LaunchAgent (macOS) service
6. Saves state locally

`zt down <name>` attempts to remove all created resources.

---

## Prerequisites

- A domain on Cloudflare
- [`cloudflared`](#installing-cloudflared) ≥ 2023.x installed and in PATH
- A Cloudflare API token with the following permissions:
  - `Account / Cloudflare Tunnel / Edit`
  - `Zone / DNS / Edit`
  - `Account / Access: Apps and Policies / Edit`

### Installing cloudflared

`zt` drives `cloudflared` but doesn't install or manage it — use your platform's
package manager so future upgrades are a normal `upgrade`/`update`, not a manual
re-download:

| Platform | Install | Upgrade |
|---|---|---|
| macOS (Homebrew) | `brew install cloudflared` | `brew upgrade cloudflared` |
| Debian/Ubuntu (apt) | see [Cloudflare's apt repo setup](https://pkg.cloudflare.com/index.html#debian-any) | `sudo apt update && sudo apt install --only-upgrade cloudflared` |
| Other Linux / binary | [Releases](https://github.com/cloudflare/cloudflared/releases) | re-download and replace the binary |
| Windows | [Releases](https://github.com/cloudflare/cloudflared/releases) | re-download and replace the binary — see [Windows support](#windows-support) |

`zt doctor` checks the installed version and, if it's outdated, prints the
right upgrade command for how it detects `cloudflared` was installed
(Homebrew/apt) or falls back to the releases link otherwise.

### Creating the API token

1. Cloudflare dashboard → **My Profile** → **API Tokens** → **Create Token**
2. Use **Custom token**, add the permissions above
3. Set Account Resources → your account
4. Set Zone Resources → your domain

---

## Install

### Option A — install script (recommended)

```bash
curl -fsSL https://raw.githubusercontent.com/casablanque-code/cfzt/main/install.sh | bash
```

Downloads the correct binary for your OS and architecture, verifies the SHA-256 checksum, and places it in `/usr/local/bin/zt`.

### Option B — go install

```bash
go install github.com/casablanque-code/cfzt/cmd/zt@latest
```

### Option C — build from source

```bash
git clone https://github.com/casablanque-code/cfzt
cd cfzt
go build -o zt ./cmd/zt
sudo mv zt /usr/local/bin/
```

### Option D — download binary

Download from [Releases](https://github.com/casablanque-code/cfzt/releases) and place in your PATH. Each release includes `.sha256` checksum files and a combined `checksums.txt`.

---

## Setup

```bash
zt init
```

You will be prompted for three values:

| Field | Where to find it |
|---|---|
| API Token | Cloudflare → My Profile → API Tokens |
| Account ID | Cloudflare dashboard → right sidebar |
| Domain | Your domain as it appears in Cloudflare (e.g. `example.com`) |

`zt init` validates the token and domain against the Cloudflare API before saving. Config is stored at `~/.zt-config.json` (mode 0600).

---

## Usage

### Bring up a tunnel

```bash
zt up <name> <port>
```

```bash
# Restrict access to specific email (Cloudflare sends OTP to that address)
zt up portainer 9000 --allow you@example.com

# Multiple allowed emails
zt up vault 8200 --allow alice@example.com --allow bob@example.com

# Auto-detect port from a running Docker container
zt up portainer --docker

# Docker + email restriction
zt up portainer --docker --allow you@example.com

# No Zero Trust gate — public access, no Access app created
zt up api 8080 --public

# ZT Access app created but bypass policy (no login required)
zt up grafana 3000

# Force TCP if QUIC is blocked by your ISP
zt up portainer 9000 --tcp
```

The service becomes available at `https://<name>.<domain>`.

The tunnel is registered as a system service and survives reboots automatically.

### Tear down a tunnel

```bash
zt down portainer
```

Stops the system service, removes local config files, deletes the DNS record, removes the Zero Trust Access app, and deletes the tunnel from Cloudflare.

### List tunnels

```bash
zt list      # or: zt ls
```

```
NAME        URL                             PORT   STATUS   MANAGED BY
portainer   https://portainer.example.com   9000   running  systemd
grafana     https://grafana.example.com     3000   stopped  pid 84291
```

### Tunnel details

```bash
zt status portainer
```

```
  portainer
  URL:        https://portainer.example.com
  Port:       9000
  Tunnel ID:  07fc193d-d05e-48eb-bb00-22be71823b14
  Managed by: systemd
  Status:     running
  Protocol:   http2 (TCP)
  Created:    2026-05-27 00:01:08
  Log:        /root/.zt/tunnels/portainer/cloudflared.log
```

### View logs

```bash
# last 50 lines
zt logs portainer

# last 100 lines
zt logs portainer -n 100

# follow (like tail -f)
zt logs portainer -f

# show logs inline with status
zt status portainer --logs
```

### Backup & restore

Export everything zt currently manages to a portable manifest:

```bash
zt export                   # writes zt.yaml in the current directory
zt export -o ~/backup/zt.yaml
```

The generated `zt.yaml` captures the intent behind each tunnel — port, protocol, access policy — but deliberately excludes credentials and Cloudflare-specific IDs. It is safe to commit to git.

```yaml
# zt.yaml — portable cfzt service manifest
# generated by `zt export` — credentials are NOT included here.
# on a new machine: run `zt init` first, then `zt apply zt.yaml`

services:
  grafana:
    port: 3000
  portainer:
    docker: true
    allow:
      - you@example.com
  vault:
    port: 8200
    protocol: quic
  api:
    port: 8080
    public: true
```

To recreate the same setup on a different machine:

```bash
# 1. on the new machine — configure credentials
zt init

# 2. apply the manifest
zt apply zt.yaml
```

`zt apply` diffs the manifest against the local state and only creates what is missing. Existing tunnels are never modified or deleted automatically:

```
⚡ Applying zt.yaml

  plan: to create: 3   skipped: 1   untracked: 0

  ~ api                 already exists — skipping

  ⚡ Bringing up grafana.example.com → localhost:3000
  ...
  🎉 Ready: https://grafana.example.com

  ✅ Done — 3 service(s) created
```

If a service exists locally but is absent from the manifest, `zt apply` reports it without touching it — remove it explicitly with `zt down <name>` if needed.

### QUIC/HTTP2 fallback watchdog

cloudflared automatically falls back from QUIC to HTTP/2 when UDP is blocked or unstable — but it never tries QUIC again on its own, even after the network recovers ([cloudflare/cloudflared#1534](https://github.com/cloudflare/cloudflared/issues/1534)). A brief UDP blip can leave a tunnel stuck on HTTP/2 indefinitely, with no automatic recovery.

`zt watchdog` runs in the background, watches each tunnel's log for the fallback, and restarts the tunnel after a backoff delay so cloudflared gets a fresh shot at QUIC. Only tunnels running with the default `protocol: auto` are affected — tunnels pinned via `--protocol` or `--tcp` are left alone, since that's a deliberate choice.

```bash
zt watchdog enable     # install as a background service, checks every 30s
zt watchdog status     # check if it's running
zt watchdog disable    # remove it
```

Restarts back off exponentially per tunnel (10 min → 20 min → ... capped at 60 min) so a tunnel with a persistently broken UDP path isn't flapped repeatedly — it still gets retried roughly once an hour rather than being abandoned.

### Health check

```bash
zt doctor
```

```
  System

  ✓  cloudflared installed
     version: cloudflared version 2024.1.0

  Cloudflare

  ✓  API token valid
  ✓  domain example.com found in Cloudflare

  Tunnel: portainer

  ✓  systemd service zt-portainer.service active
  ✓  local service on port 9000 reachable
  ✓  DNS resolves portainer.example.com
  ✓  Cloudflare tunnel exists

  ✓ all checks passed
```

---

## Flags

### `zt export`

| Flag | Description |
|---|---|
| `-o <path>` | Output path (default: `zt.yaml` in current directory) |

### `zt apply`

`zt apply <file>` takes no additional flags. It reads the manifest at `<file>` and creates any missing services.

### `zt watchdog`

| Subcommand | Description |
|---|---|
| `enable` | Install and start the watchdog as a background service |
| `disable` | Stop and remove the watchdog service |
| `status` | Show whether the watchdog is running |

### `zt up`

| Flag | Description |
|---|---|
| `--allow <email>` | Restrict access to this email via Cloudflare Access (repeatable) |
| `--public` | No Zero Trust gate — skip Access app entirely |
| `--docker` | Auto-detect port from a running Docker container with this name |
| `--tcp` | Force TCP (http2) — use if QUIC/UDP is blocked by your ISP |
| `--protocol <proto>` | Protocol: `auto` (default), `quic`, `http2` |

### `zt logs`

| Flag | Description |
|---|---|
| `-n <lines>` | Number of lines to show (default: 50) |
| `-f` | Follow log output |

### `zt status`

| Flag     | Description                    |
| -------- | ------------------------------- |
| `--logs` | Show recent log output inline   |

`zt status <name>` and `zt <name> status` are equivalent — same for `logs`, `restart`, and `down`.

---

## Shell completion

[#shell-completion](#shell-completion)

`zt` uses [Cobra](https://github.com/spf13/cobra), which ships tab-completion out of the box:

```
# bash (current shell)
source <(zt completion bash)

# bash (persist)
zt completion bash | sudo tee /etc/bash_completion.d/zt

# zsh
zt completion zsh > "${fpath[1]}/_zt"

# fish
zt completion fish > ~/.config/fish/completions/zt.fish
```

Run `zt completion --help` for details per shell.

---

## Version

[#version](#version)

```
zt --version
```

There is currently no `zt update` command — `zt` is a single static binary with no
auto-updater. To upgrade, re-run the install script or `go install` (see
[Install](#install)), which simply overwrites the existing binary in place:

```
curl -fsSL https://raw.githubusercontent.com/casablanque-code/cfzt/main/install.sh | bash
```

---

## Windows support

[#windows-support](#windows-support)

`zt up`/`down`/`restart`/`status`/`logs`/`doctor` work on Windows — cloudflared
runs directly as a tracked process (PID mode), same fallback mode Linux/macOS
use when systemd/launchd aren't available.

**Not yet implemented on Windows:**

- No persistent service — cloudflared won't survive a reboot or auto-restart
  after a crash. `zt up` will tell you this explicitly (`(no auto-restart)`)
  when it happens.
- `zt watchdog` is unavailable.

### Installing `zt` on Windows

**PowerShell (recommended)** — downloads the latest release, verifies its
checksum, and adds it to your user `PATH` safely:

```powershell
$dest = "$env:LOCALAPPDATA\zt"
New-Item -ItemType Directory -Force -Path $dest | Out-Null

$exe = "$dest\zt.exe"
Invoke-WebRequest "https://github.com/casablanque-code/cfzt/releases/latest/download/zt-windows-amd64.exe" -OutFile $exe
Invoke-WebRequest "https://github.com/casablanque-code/cfzt/releases/latest/download/zt-windows-amd64.exe.sha256" -OutFile "$exe.sha256"

$expected = (Get-Content "$exe.sha256").Split(" ")[0]
$actual = (Get-FileHash $exe -Algorithm SHA256).Hash.ToLower()
if ($expected -ne $actual) { throw "checksum mismatch — do not run this binary" }

$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($userPath -notlike "*$dest*") {
    [Environment]::SetEnvironmentVariable("Path", "$userPath;$dest", "User")
}

Write-Host "Installed. Open a new terminal and run: zt init"
```

`zt.exe` is a console app — double-clicking it in Explorer will show a
"you need to open cmd.exe" prompt. That's expected; always run it from a
terminal (cmd, PowerShell, Windows Terminal).

**Don't use `setx PATH ...` to add it to PATH by hand** — `setx` truncates
anything over 1024 characters *silently*, and on a dev machine with a few
toolchains already installed your PATH is almost certainly already longer
than that. It will corrupt your PATH rather than append to it. The script
above uses `[Environment]::SetEnvironmentVariable`, which doesn't have that
limit — but if your PATH still looks wrong afterwards (check with
`echo %PATH%` in a **new** cmd window — env var changes don't apply to
already-open terminals), fix it via the GUI instead: `Win`+`R` → `sysdm.cpl`
→ **Advanced** → **Environment Variables**, rather than any more `setx`.

---

## File layout

```
~/.zt-config.json                      # credentials (0600)
~/.zt-state.json                       # tunnel state (0600)
~/.zt/tunnels/<name>/
    config.yml                         # cloudflared config
    <tunnel-id>.json                   # tunnel credentials
    cloudflared.log                    # cloudflared process log

~/.config/systemd/user/
    zt-<name>.service                  # systemd unit (Linux)

~/Library/LaunchAgents/
    com.zt.<name>.plist                # LaunchAgent (macOS)
```

---

## Troubleshooting

**`cloudflared not found in PATH`**
Install it: https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/downloads/

**`unsupported cloudflared version`**
Your cloudflared is older than 2023.x. Upgrade using the link above.

**`502 Bad Gateway` after `zt up`**
The tunnel is up but the local service is not running or not listening on the specified port.
```bash
curl http://localhost:<port>
zt logs <name>
```

**`502 Bad Gateway` even though `zt status` shows the QUIC connection is up**
`zt status` reflects what cloudflared's connectivity pre-check and connection
registration reported - but that pre-check only opens a small test UDP
connection, not real traffic. On some networks the pre-check passes (QUIC
control packets get through fine) while actual response data gets silently
dropped due to UDP fragmentation/MTU issues on the path, causing request
timeouts that surface as 502s. This is not something `zt` or the watchdog
can detect or fix automatically - if you're seeing 502s with an apparently
healthy QUIC connection, force TCP:
```bash
zt down <name> && zt up <name> <port> --tcp
```

**Tunnel shows `stopped` in `zt ls`**
```bash
systemctl --user status zt-<name>
zt logs <name>
```
If the service crashed, restart it:
```bash
systemctl --user restart zt-<name>
```
Or tear down and recreate:
```bash
zt down <name> && zt up <name> <port>
```

**`tunnel already exists`**
Run `zt down <name>` first. If the tunnel is stale on Cloudflare's side (e.g. after a failed previous run), `zt up` detects and removes it automatically before creating a new one.

**`zone not found for domain`**
Make sure the domain is added to Cloudflare and the API token has `Zone / DNS / Edit` permission.

**Authentication error on Access app creation**
The API token is missing `Account / Access: Apps and Policies / Edit`. Edit the token in the Cloudflare dashboard.

**DNS record conflict**
`zt up` uses upsert — it removes any existing A, AAAA, or CNAME record with the same name before creating the tunnel CNAME. No manual cleanup needed.

**Run `zt doctor` first**
Most issues are diagnosed automatically:
```bash
zt doctor
```

---

## License

MIT
