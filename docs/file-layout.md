# File layout

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

Task Scheduler > zt-<name>              # scheduled task (Windows, no on-disk file)
```
