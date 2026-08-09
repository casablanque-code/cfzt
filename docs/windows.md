# Windows support

`zt up` installs a Task Scheduler task (`zt-<name>`, logon trigger,
restart-on-failure) — the same auto-start/auto-restart guarantee
systemd/launchd give on Linux/macOS, and `zt down`/`restart`/`status`/`logs`/
`doctor` all work against it. `zt watchdog` is also available, running as its
own `zt-watchdog` task.

The task runs under your own logon session with no stored credentials, so
`zt up` never needs admin rights or a password prompt — same permission
model as `systemctl --user` / a per-user LaunchAgent. The trade-off is the
same one those have too: it only starts once *you* log in, not at machine
boot before any user session exists.

If `cloudflared` isn't on `PATH` (or task creation fails for any other
reason), `zt up` falls back to running it as a directly tracked process —
you'll see `(no auto-restart)` in the output when that happens.

`zt down`/`zt watchdog disable` end the task via `schtasks /end` and wait
for it to actually leave the Running state before deleting its
registration — `/end` only requests termination and returns immediately,
so without the wait a slow-to-die `cloudflared` process could still be
alive and holding its port right after teardown reported success. If it's
still running once the (short) wait is up, teardown proceeds anyway and
prints a warning rather than blocking indefinitely on a possibly-stuck
process.

Windows install: [Scoop](../README.md#option-f--scoop-windows)
(recommended, self-updating). A WinGet package is in progress — see the
issue tracker for status.

## Windows-specific gotchas

**Don't use `setx PATH ...` to add `zt` to PATH by hand** — `setx`
truncates anything over 1024 characters *silently*, and on a dev machine
with a few toolchains already installed your PATH is almost certainly
already longer than that. It will corrupt your PATH rather than append to
it. The install script uses `[Environment]::SetEnvironmentVariable`, which
doesn't have that limit — but if your PATH still looks wrong afterwards
(check with `echo %PATH%` in a **new** cmd window — env var changes don't
apply to already-open terminals), fix it via the GUI instead: `Win`+`R` →
`sysdm.cpl` → **Advanced** → **Environment Variables**, rather than any
more `setx`.

`zt.exe` is a console app — double-clicking it in Explorer will show a
"you need to open cmd.exe" prompt. That's expected; always run it from a
terminal (cmd, PowerShell, Windows Terminal).

**Task doesn't start / `schtasks` shows the task but it's not running**
The `zt-<name>` task only fires on logon to *your* account — it won't
start before you sign in, and it won't run under a different user's
session. Sign in and check:
```bash
schtasks /query /tn zt-<name> /v
zt logs <name>
```
