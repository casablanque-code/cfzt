# QUIC/HTTP2 fallback watchdog

cloudflared automatically falls back from QUIC to HTTP/2 when UDP is
blocked or unstable — but it never tries QUIC again on its own, even after
the network recovers
([cloudflare/cloudflared#1534](https://github.com/cloudflare/cloudflared/issues/1534)).
A brief UDP blip can leave a tunnel stuck on HTTP/2 indefinitely, with no
automatic recovery.

`zt watchdog` runs in the background, watches each tunnel's log for the
fallback, and restarts the tunnel after a backoff delay so cloudflared gets
a fresh shot at QUIC. Only tunnels running with the default `protocol:
auto` are affected — tunnels pinned via `--protocol` or `--tcp` are left
alone, since that's a deliberate choice.

```bash
zt watchdog enable     # install as a background service, checks every 30s
zt watchdog status     # check if it's running
zt watchdog disable    # remove it
```

Restarts back off exponentially per tunnel (10 min → 20 min → ... capped
at 60 min) so a tunnel with a persistently broken UDP path isn't flapped
repeatedly — it still gets retried roughly once an hour rather than being
abandoned.

**Known limitation:** if a tunnel is restarted manually (`zt restart
<name>`) at the same moment the watchdog independently decides to restart
it for a fallback, the two restarts can race. This is a known open issue,
not yet fixed.
