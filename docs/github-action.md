# GitHub Action

`casablanque-code/cfzt` is also a composite action wrapping `zt up`/`zt
down` for the PR-preview flow (see the main [README](../README.md#use-cases)),
plus a GitHub Deployment + Deployment Status so the preview URL shows up in
the PR's own UI, not just build logs.

```yaml
name: PR preview
on:
  pull_request:
    types: [opened, synchronize, reopened, closed]

permissions:
  deployments: write   # needed for the GitHub Deployments UI integration

jobs:
  preview:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        if: github.event.action != 'closed'
      # ... build your image / start the container on localhost:3000 ...

      - uses: casablanque-code/cfzt@v0.10.0
        if: github.event.action != 'closed'
        with:
          mode: up
          name: pr-${{ github.event.number }}
          port: '3000'
          docker: 'true'
          public: 'true'   # or use `allow:` to restrict to your team
          domain: example.com
          cloudflare-api-token: ${{ secrets.CLOUDFLARE_API_TOKEN }}
          cloudflare-account-id: ${{ secrets.CLOUDFLARE_ACCOUNT_ID }}

      - uses: casablanque-code/cfzt@v0.10.0
        if: github.event.action == 'closed'
        with:
          mode: down
          name: pr-${{ github.event.number }}
          domain: example.com
          cloudflare-api-token: ${{ secrets.CLOUDFLARE_API_TOKEN }}
          cloudflare-account-id: ${{ secrets.CLOUDFLARE_ACCOUNT_ID }}
```

Pin to a tagged release (`@v0.10.0`) rather than `@main` — same
reproducibility reasoning as the `cloudflared-version`/`cfzt-version`
inputs below.

## Notes

- **`name` must match** between the `up` and `down` calls — it's the
  tunnel name, the hostname prefix, and what `down` resolves against
  Cloudflare by name (see below).
- **State across runs:** `up` and `down` normally run in entirely
  separate, unrelated jobs — possibly weeks apart, on unrelated
  ephemeral runners — so there's no local state file for `down` to find
  the way there would be if the same machine had run `zt up`. The action
  calls `zt down --remote`, which resolves the tunnel directly from
  Cloudflare by name instead of requiring it in local state. This is
  opt-in on the CLI itself (a plain `zt down` still requires local
  state) because a Cloudflare Tunnel has no "created by zt" marker the
  way a zt-managed DNS record does — resolving by name alone means
  whatever tunnel exists under that exact name gets deleted, which is
  exactly what a CI job tearing down its own preview wants and exactly
  what a mistyped `zt down` on your own machine doesn't. See
  [SECURITY.md](../SECURITY.md) for the full risk note on `--remote`.
- **Deployment matching:** the `down` step marks GitHub Deployments
  inactive by matching on `task: "cfzt:<name>"`, set at creation time —
  not just `environment`, since `environment` defaults to `name` but is
  user-overridable and can be reused across multiple previews.
- `permissions: deployments: write` is required on the calling workflow
  for the Deployments UI integration — a composite action can't grant
  itself permissions. Set `create-deployment: 'false'` to skip that part
  entirely and just get the tunnel.
- **Reproducibility:** `cloudflared-version` and `cfzt-version` inputs
  pin the exact `cloudflared`/`zt` builds installed by the action,
  instead of always tracking `latest`/`main`. Leaving them unset keeps
  the simpler default behavior, with a workflow warning nudging toward
  pinning.
- If recording the GitHub Deployment fails after `zt up` already
  succeeded, the action runs `zt down --remote` as a cleanup step so the
  tunnel isn't left orphaned.
- `Validate inputs` checks for `gh`/`jq` on `PATH` up front with an
  actionable error if either is missing.
- The Cloudflare API token written to `~/.zt-config.json` for the
  duration of the run is removed at the end of it (only if the run
  itself created the file).

See [`action.yml`](../action.yml) for the full list of inputs.
