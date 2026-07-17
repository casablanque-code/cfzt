# Security Policy

## Supported Versions

Only the latest release receives security fixes.

| Version | Supported |
|---------|-----------|
| 0.6.x   | ✅        |
| < 0.4   | ❌        |

## Reporting a Vulnerability

**Please do not open a public GitHub issue for security vulnerabilities.**

Email: casablanque@proton.me  
Response time: within 72 hours

Include in your report:
- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Suggested fix (if any)

## Scope

Things we care about most:

- API token exposure or leakage (stored in `~/.zt-config.json`)
- Privilege escalation via systemd service or cloudflared config
- Unintended public exposure of tunnels (bypass of Zero Trust policy)
- Supply chain issues in CI/CD (GitHub Actions, dependencies)

## Out of Scope

- Cloudflare platform vulnerabilities — report those to [Cloudflare directly](https://www.cloudflare.com/disclosure/)
- Issues in `cloudflared` itself — report to [cloudflare/cloudflared](https://github.com/cloudflare/cloudflared)

## Security Notes

- Credentials are stored at `~/.zt-config.json` with mode `0600`
- Tunnel credentials are stored at `~/.zt/tunnels/<name>/<id>.json` with mode `0600`
- cfzt never transmits credentials anywhere except the Cloudflare API over HTTPS
- The Gitleaks CI scan checks all commits for accidentally leaked secrets
