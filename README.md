# zt — Zero Trust tunnel manager

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
- [`cloudflared`](https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/downloads/) ≥ 2023.x installed and in PATH
- A Cloudflare API token with the following permissions:
  - `Account / Cloudflare Tunnel / Edit`
  - `Zone / DNS / Edit`
  - `Account / Access: Apps and Policies / Edit`

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

### `zt up`

| Flag | Description |
|---|---|
| `--allow <email>` | Restrict access to this email via Cloudflare Access (repeatable) |
| `--public` | No Zero Trust gate — skip Access app entirely |
| `--docker` | Auto-detect port from a running Docker container with this name |

### `zt logs`

| Flag | Description |
|---|---|
| `-n <lines>` | Number of lines to show (default: 50) |
| `-f` | Follow log output |

### `zt status`

| Flag | Description |
|---|---|
| `--logs` | Show recent log output inline |

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
