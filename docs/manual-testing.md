# Manual testing

These scenarios need a real Docker daemon and a real Cloudflare account
with a domain — they're not run in CI and there's no automated e2e suite
for them yet. Run through this checklist after any change touching
tunnel lifecycle, Docker port detection, the manifest, or Windows service
management, and before tagging a release that touches those areas.

Replace `example.com` with your own domain and `you@example.com` with
your own email throughout.

## 1. Basic lifecycle

```bash
zt up smoketest 3000 --allow you@example.com
zt status smoketest        # running, correct port/hostname
zt list                    # shows smoketest
zt logs smoketest -n 20
curl -I https://smoketest.example.com   # expect a Cloudflare Access redirect (302), not 502
zt down smoketest
zt list                    # smoketest gone
```

Expected: each step succeeds; `zt down` removes the DNS record, Access
app, and tunnel (check the Cloudflare dashboard if in doubt).

## 2. Docker port detection — determinism and `--container-port`

```bash
docker run -d --name multitest -p 18080:80 -p 18443:443 nginx

zt up multitest --docker --allow you@example.com
zt status multitest        # expect port 18080 (lowest published TCP port, 80/tcp)
zt down multitest

zt up multitest --docker --container-port 443 --allow you@example.com
zt status multitest        # expect port 18443
zt down multitest

docker rm -f multitest
```

Expected: the plain `--docker` run always picks the same (lowest) port
across repeated up/down cycles — run the first block 3-4 times in a row
to confirm it never flips. `--container-port 443` always picks 18443.

```bash
docker run -d --name singleport -p 19090:80 nginx
zt up singleport --docker --container-port 443 --allow you@example.com
docker rm -f singleport
```

Expected: a clear error naming the missing container port (`no published
TCP binding for container port 443`), not a silent fallback to 19090 and
not a crash.

## 3. Export / apply round-trip (same machine, idempotency)

```bash
zt up sanity 3000 --allow you@example.com
zt export -o /tmp/zt.yaml
zt apply /tmp/zt.yaml       # re-applying the same manifest on the same machine
zt list                     # still exactly one "sanity", same tunnel ID as before
zt down sanity
```

Expected: the second `zt apply` prints `already exists — skipping` for
`sanity` and does **not** create a second tunnel or overwrite the
existing state entry.

```bash
docker run -d --name webtest -p 20080:80 -p 20443:443 nginx
cat > /tmp/manifest.yaml <<'EOF'
services:
  webtest:
    docker: true
    container_port: '443'
    public: true
EOF
zt apply /tmp/manifest.yaml
zt status webtest           # expect 20443
zt down webtest
docker rm -f webtest
```

```bash
cat > /tmp/bad-manifest.yaml <<'EOF'
services:
  bad:
    port: 3000
    container_port: '443'
EOF
zt apply /tmp/bad-manifest.yaml
```

Expected: `container_port` round-trips correctly through export/apply,
and the manifest without `docker: true` fails validation
(`'container_port' requires 'docker: true'`) rather than silently
ignoring the field.

## 4. DNS conflict and `--force`

```bash
# manually create an A or CNAME record for conflict.example.com in the
# Cloudflare dashboard first, pointing anywhere that isn't a zt tunnel
zt up conflict 3000 --allow you@example.com
# expect: refuses with "existing DNS record found ... refusing to replace it"
zt up conflict 3000 --allow you@example.com --force
# expect: succeeds, replaces the record
zt down conflict
```

## 5. Watchdog

```bash
zt up watchtest 3000 --allow you@example.com
zt watchdog enable
zt watchdog status          # running
# block UDP/443 outbound temporarily (e.g. a local firewall rule) to
# force a QUIC->HTTP2 fallback, or just leave it running for a while on
# a network you know drops UDP sometimes
zt logs watchtest -f        # watch for the fallback + restart
zt watchdog disable
zt down watchtest
```

Expected: the watchdog restarts the tunnel after the backoff delay
following a fallback, and `watchdog disable` cleanly stops and removes
its service.

## 6. Orphaned process recovery (Windows)

Simulates the scenario the `wait for task to actually stop before
deleting it` fix addresses.

```powershell
zt up patchbook 3000 --allow you@example.com
# note the cloudflared.exe PID from `tasklist`, then simulate an
# unusually slow shutdown or just kill the scheduled task registration
# out from under it directly, e.g.:
schtasks /delete /tn zt-patchbook /f    # deletes registration without /end
zt list                                 # patchbook likely still shows, or is now untracked
tasklist | findstr /i cloudflared       # process may still be alive
```

If you land in this state (task gone from Task Scheduler, process still
running, `zt down patchbook` fails with "not found in local state" or
similar):

```powershell
tasklist | findstr /i "cloudflared zt.exe"
netstat -ano | findstr <PID>            # confirm what it's holding
taskkill /F /PID <PID>
zt down patchbook --remote               # resolves and tears down by
                                          # name directly from Cloudflare,
                                          # bypassing local state entirely
tasklist | findstr /i "cloudflared zt.exe"   # expect empty
schtasks /query /tn zt-patchbook 2>&1        # expect "cannot find"
```

Expected: `--remote` finds and deletes the Cloudflare-side tunnel (and
DNS/Access if still present) even with no usable local state, and the
orphaned process/task registration can be manually cleared with the
commands above. On a clean `zt up`/`zt down` cycle post-fix, this
manual cleanup should not be necessary — the `waitForTaskStopped` poll
should mean `zt down` reports a warning (not silence) if the process
was slow to exit, rather than leaving no trace at all.

## 7. GitHub Action — deployment matching

In a real PR against a repo using the action:

```bash
# after "up" runs (PR opened/synchronize)
gh api repos/<owner>/<repo>/deployments --jq '.[] | {id, task, environment}'
# expect an entry with task == "cfzt:pr-<number>"

# after the PR is closed (down runs)
gh api repos/<owner>/<repo>/deployments --jq '.[] | {id, task, state: .statuses_url}'
```

Expected: closing the PR marks only the deployment(s) with that PR's
`task` value inactive — if you have a second, unrelated preview sharing
the same `environment` input value, its deployment must be unaffected.
