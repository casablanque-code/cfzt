# Troubleshooting

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
zt doctor
zt logs <name>
```
Or check the service manager directly:
```bash
systemctl --user status zt-<name>      # Linux
launchctl list com.zt.<name>           # macOS
schtasks /query /tn zt-<name> /v       # Windows
```
If the service crashed, restart it:
```bash
zt restart <name>
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
`zt up` upserts the tunnel's own CNAME automatically — if a previous zt tunnel left a stale CNAME to `*.cfargotunnel.com` for the same hostname, it's replaced without asking. If the hostname already has an A, AAAA, or CNAME record that zt didn't create (e.g. it's pointing somewhere else entirely), `zt up` refuses and prints `existing DNS record found ... refusing to replace it`. Re-run with `--force` to replace it anyway, or free up the hostname first.

**Task doesn't start on Windows / `schtasks` shows the task but it's not running**
See [Windows support](windows.md).

**Run `zt doctor` first**
Most issues are diagnosed automatically:
```bash
zt doctor
```
