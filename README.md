# npm-vet

A transparent npm install interceptor that vets packages for typosquatting, malicious scripts, obfuscated code, and other supply chain threats before they reach your machine.

**npm-vet** is a single-binary CLI tool that sits in front of `npm` and vets every package before installation. It detects typosquatting, suspicious postinstall scripts, obfuscated code, network requests during install, maintainer takeovers, and brand-new low-download packages — all without accounts, API keys, or external dependencies.

It works two ways: as a **transparent interceptor** that replaces `npm` in your PATH (catching installs from scripts, agents, and IDE extensions automatically), and as a **CI/CD audit tool** that scans your `package.json` and exits non-zero when it finds something wrong.

## Install

Requires Go 1.21+.

```bash
git clone https://github.com/matthewblackburn/npm-vet.git
cd npm-vet
make install
```

This builds the binary, installs it to `~/.npm-vet/`, creates an `npm` shim, and adds it to your PATH. Open a new terminal to activate.

To uninstall:

```bash
make uninstall
```

## How it works

After installation, **every** `npm` call is transparently intercepted:

```bash
npm install express        # vetted automatically, then installed
npm install expresss       # blocked — typosquat of "express"
npm test                   # passed straight through to real npm
npm run build              # passed straight through to real npm
```

When an install command is detected, npm-vet:

1. Parses the packages being installed (from CLI args or `package.json`)
2. Fetches metadata and download stats from the npm registry in parallel
3. Downloads and scans tarballs for code-level analysis
4. Runs all 6 security analyzers concurrently
5. Reports findings and prompts you to continue or abort
6. If approved, delegates to the real `npm` to perform the actual install

Non-install commands (`npm test`, `npm run build`, etc.) pass through to npm instantly with zero overhead.

## Security checks

| Analyzer | What it detects | Severity |
|---|---|---|
| **Postinstall scripts** | `curl \| sh`, `wget`, `node -e`, remote URLs in lifecycle scripts | Critical |
| **Obfuscated code** | `eval()`, `new Function()`, long encoded strings, hex/unicode density | Critical when combined |
| **Network requests** | `http`/`net`/`dns`/`fetch` imports in install scripts | Critical |
| **Maintainer changes** | New, removed, or fully replaced maintainers vs local cache | Critical on full replacement |
| **Typosquatting** | Package names within edit distance 1-2 of top 500 popular packages | Critical if distance=1 from top 100 |
| **Low downloads** | Packages with <100 weekly downloads, especially if published within 7 days | Critical if new + low |

## CI/CD usage

npm-vet includes an audit mode designed for CI/CD pipelines:

```bash
# Basic audit — exits 1 on critical findings
npm-vet audit

# JSON output for machine parsing
npm-vet audit --json

# SARIF output for GitHub Advanced Security / code scanning
npm-vet audit --sarif

# Fail on warnings too (default: fail on critical only)
npm-vet audit --fail-on=warning

# Scan a specific directory
npm-vet audit --dir=./packages/frontend
```

### Exit codes

| Code | Meaning |
|---|---|
| 0 | No findings above threshold |
| 1 | Findings at or above `--fail-on` severity |
| 2 | Runtime error (e.g., missing package.json) |

### GitHub Actions example

```yaml
jobs:
  security:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.21'
      - run: go install github.com/matthewblackburn/npm-vet@latest
      - run: npm-vet audit --fail-on=warning
```

## Configuration

Create a `.npm-vetrc` file in your project root (or `~/.npm-vetrc` for global config):

```json
{
  "allowlist": ["lodash", "express", "@types/*"],
  "mode": "block",
  "fail_on": "critical",
  "analyzers": {
    "postinstall": true,
    "obfuscation": true,
    "network": true,
    "maintainer": true,
    "typosquat": true,
    "downloads": true
  },
  "thresholds": {
    "min_weekly_downloads": 100,
    "max_typosquat_distance": 2,
    "max_string_length": 1000,
    "max_tarball_size_mb": 5
  }
}
```

The allowlist supports glob patterns (`@types/*` matches `@types/node`, `@types/react`, etc.).

### AI agent safety

npm-vet uses TTY detection to distinguish between humans and AI agents. When security findings are detected:

- **Human in a terminal** → shown findings, prompted to type `INSTALL` to proceed
- **AI agent or script** (non-interactive stdin) → **hard blocked**, exit code 1, no prompt, no way to override

This is bulletproof — AI agents (Claude Code, Copilot, Cursor, etc.) run commands via piped subprocesses where stdin is not a TTY. They cannot allocate a real terminal. The agent will see the findings and be told to ask the user:

```
npm-vet: BLOCKED — non-interactive session detected (AI agent or script).
npm-vet: Security findings require human review. Run this install manually in your terminal,
npm-vet: or add the package to your .npm-vetrc allowlist.
```

If the package is legitimate, add it to your `.npm-vetrc` allowlist and the agent can install it freely on subsequent runs.

## Accessing npm-vet commands when installed as shim

When npm-vet is installed as the `npm` shim, access its subcommands via `npm vet`:

```bash
npm vet audit --json       # run audit
npm vet version            # show npm-vet version
npm vet teardown           # disable interception
```

## Environment variables

| Variable | Description |
|---|---|
| `NPM_VET_NPM_PATH` | Override path to real npm binary |
| `CI` | When set, disables colors and interactive prompts |

## How is this different from...

| Tool | Requires account? | Intercepts npm? | CI/CD mode? | Self-contained? |
|---|---|---|---|---|
| **npm-vet** | No | Yes (transparent shim) | Yes | Single binary, no deps |
| Socket CLI | Yes | Yes | Yes | Requires account + API |
| npq | No | No (separate command) | No | Requires Node + Snyk |
| npm audit | No | No (post-install) | Yes | CVE database only |

## License

MIT
