# zt — Zero Trust tunnel manager

One command to expose a local service through Cloudflare Zero Trust.

```
zt up portainer 9000 --allow you@example.com
# → https://portainer.yourdomain.com  (ZT-protected, running in ~15s)
```

## What it does

`zt up <name> <port>` automatically:

1. Creates a Cloudflare Tunnel
2. Configures ingress rules
3. Upserts a CNAME DNS record
4. Creates a Zero Trust Access application
5. Starts `cloudflared` in the background
6. Saves state locally

`zt down <name>` reverses all of the above.

---

## Prerequisites

- A domain on Cloudflare
- [`cloudflared`](https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/downloads/) installed and in PATH
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

Detects OS and architecture, downloads the correct binary from the latest release, places it in `/usr/local/bin/zt`.

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

Download from [Releases](https://github.com/casablanque-code/cfzt/releases) and place in your PATH.

---

## Setup

```bash
zt init
```

You'll be prompted for:

| Field | Where to find it |
|---|---|
| API Token | Cloudflare → My Profile → API Tokens |
| Account ID | Cloudflare dashboard → right sidebar |
| Domain | Your domain as it appears in Cloudflare (e.g. `example.com`) |

Config is saved to `~/.zt-config.json` (mode 0600).

---

## Usage

### Bring up a tunnel

```bash
zt up <name> <port>
```

```bash
# Zero Trust protected — prompts email login via Cloudflare Access
zt up portainer 9000 --allow you@example.com

# Multiple allowed emails
zt up vault 8200 --allow alice@example.com --allow bob@example.com

# Access app with bypass policy (no login required, but still proxied through CF)
zt up grafana 3000

# Completely public, no Access app created
zt up api 8080 --public
```

Service becomes available at `https://<name>.<domain>`.

### Tear down a tunnel

```bash
zt down portainer
```

Stops the cloudflared process, removes the DNS record, deletes the tunnel and Access app from Cloudflare.

### List tunnels

```bash
zt list      # or: zt ls
```

```
NAME        URL                             PORT   STATUS    PID
portainer   https://portainer.example.com   9000   running   17423
grafana     https://grafana.example.com     3000   stopped   -
```

### Tunnel details

```bash
zt status portainer
```

---

## Flags

### `zt up`

| Flag | Description |
|---|---|
| `--public` | No Zero Trust gate — skip Access app entirely |
| `--allow <email>` | Restrict access to this email, repeatable |

---

## File layout

```
~/.zt-config.json                  # credentials (0600)
~/.zt-state.json                   # tunnel state (0600)
~/.zt/tunnels/<name>/
    config.yml                     # cloudflared config
    <tunnel-id>.json               # tunnel credentials
    cloudflared.log                # cloudflared process log
```

---

## Troubleshooting

**`cloudflared not found in PATH`**
Install it: https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/downloads/

**`502 Bad Gateway` after `zt up`**
The tunnel is up but the local service isn't running or isn't listening on the specified port. Check with `curl http://localhost:<port>` and look at the log:
```bash
tail -f ~/.zt/tunnels/<name>/cloudflared.log
```

**Tunnel shows `stopped` but URL still works**
The cloudflared process was restarted outside of zt. Run `zt down <name>` + `zt up <name> <port>` to resync state.

**`tunnel already exists`**
Run `zt down <name>` first, or check `zt list`. If the tunnel is stale on Cloudflare's side, `zt up` will clean it up automatically.

**`zone not found for domain`**
Make sure the domain is added to Cloudflare and the API token has Zone / DNS / Edit permission.

**Authentication error on Access app creation**
The API token is missing `Account / Access: Apps and Policies / Edit` permission. Edit the token in Cloudflare dashboard and add it.

---

## License

MIT
