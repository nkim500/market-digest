# Security Policy

## Reporting a Vulnerability

**Do NOT open a public issue for security vulnerabilities.**

Instead, please email **nick.mj.kim@gmail.com** with:

1. Description of the vulnerability
2. Steps to reproduce
3. Potential impact
4. Suggested fix (if any)

You will receive a response within 72 hours. We will work with you to understand and address the issue before any public disclosure.

## Scope

Security issues in the following are in scope:

- **Jobs binary** (`jobs/`, `internal/`) — command injection, path traversal, SSRF in HTTP fetchers
- **Dashboard** (`dashboard/`) — any Go binary vulnerabilities
- **SQLite handling** — SQL injection, DB corruption on adversarial input
- **Configuration** — secrets exposure, unsafe defaults in `config/sources.yml`

## Out of Scope

- Issues in third-party dependencies (report upstream)
- Issues requiring physical access to the user's machine
- Social engineering attacks
- market-digest is a local tool — there is no hosted service to attack

## Disclosure Policy

We follow coordinated disclosure. Once a fix is released, we will credit the reporter (unless they prefer anonymity) in the release notes.
