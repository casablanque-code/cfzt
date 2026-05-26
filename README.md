# zt — Zero Trust tunnel manager

One command to expose a local service through Cloudflare Zero Trust.

```
zt up grafana 3000
# → https://grafana.yourdomain.com  (ZT-protected, running in 15s)
```

## What it does

`zt up <name> <port>` automatically:

1. Creates a Cloudflare Tunnel
2. Configures ingress rules
3. Creates a CNAME DNS record
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

### Option A — go install

```bash
go install github.com/casablanque-code/cfzt/cmd@latest
```

### Option B — build from source

```bash
git clone https://github.com/casablanque-code/cfzt
cd cfzt
go build -o zt ./cmd
sudo mv zt /usr/local/bin/
```

### Option C — download binary

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
# Expose Grafana (Zero Trust protected, bypass policy by default)
zt up grafana 3000

# Expose an API publicly (no ZT gate)
zt up api 8080 --public

# Restrict to specific emails
zt up vault 8200 --allow alice@example.com --allow bob@example.com
```

Service becomes available at `https://<name>.<domain>`.

### Tear down a tunnel

```bash
zt down grafana
```

Stops the process, removes DNS record, deletes the tunnel from Cloudflare.

### List tunnels

```bash
zt list      # or: zt ls
```

```
NAME      URL                           PORT   STATUS    PID
grafana   https://grafana.example.com   3000   running   84291
vault     https://vault.example.com     8200   stopped   -
```

### Tunnel details

```bash
zt status grafana
```

---

## Flags

### `zt up`

| Flag | Description |
|---|---|
| `--public` | Skip Zero Trust gate (public access) |
| `--allow <email>` | Restrict to email(s), repeatable |

---

## File layout

```
~/.zt-config.json              # credentials (0600)
~/.zt-state.json               # tunnel state (0600)
~/.zt/tunnels/<name>/
    config.yml                 # cloudflared config
    <tunnel-id>.json           # tunnel credentials
    cloudflared.log            # process log
```

---

## Troubleshooting

**`cloudflared not found in PATH`**
Install it: https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/downloads/

**Tunnel shows `stopped` but service is still accessible**
The cloudflared process may have been restarted externally. Run `zt down` + `zt up` to resync state.

**`zone not found for domain`**
Make sure the domain is added to Cloudflare and the API token has Zone / DNS / Edit permission.

**`tunnel already exists`**
Run `zt down <name>` first, or check `zt list`.

---

## License

MIT
