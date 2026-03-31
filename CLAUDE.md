# CLAUDE.md

## Project overview

npm-vet is a single-binary Go CLI tool that intercepts `npm install` and vets packages for supply chain threats before installation. It works as a transparent shim (replacing `npm` in PATH) and as a standalone CI/CD audit tool.

## Build & test

```bash
go build -o npm-vet .         # Build
go test ./...                  # Run all tests
make install                   # Build + install to ~/.npm-vet/ + activate shim
make uninstall                 # Remove binary + shim
```

## Architecture

- **`main.go`** — Entry point. Routes to wrapper mode (default), audit, setup, or teardown based on args and invocation name (`npm` vs `npm-vet`).
- **`internal/cmd/`** — Command handlers: `wrapper.go` (intercept installs), `audit.go` (CI/CD scan), `setup.go` (install shim).
- **`internal/parser/`** — Parse npm CLI args and package.json/lockfiles.
- **`internal/registry/`** — npm registry HTTP client with caching. Types for metadata, downloads, tarballs.
- **`internal/pipeline/`** — Shared `Run(packages, cfg) → []Finding` orchestrator used by both wrapper and audit modes. Parallel fetch + analyze via errgroup.
- **`internal/analyzer/`** — 6 analyzers implementing the `Analyzer` interface: postinstall, obfuscation, network, maintainer, typosquat, downloads.
- **`internal/reporter/`** — Console (colored), JSON, and SARIF output formatters.
- **`internal/config/`** — `.npm-vetrc` config file loading with defaults.
- **`internal/delegator/`** — Build-tagged (`exec_unix.go`, `exec_windows.go`) npm delegation. Finds real npm in PATH, skipping self.
- **`internal/npmvet/`** — Core types: `Finding`, `Severity`, `PackageSpec`.

## Key conventions

- **Single external dep**: `golang.org/x/sync/errgroup`. Everything else is stdlib.
- **Tests live alongside code**: `*_test.go` files in each package (Go convention).
- **Graceful degradation**: HTTP timeouts (10s per request, 60s global). Failed fetches produce warnings, not errors.
- **Tarball size cap**: 5MB. Larger packages skip tarball-based analyzers with an info finding.
- **Maintainer analyzer**: Silent on first encounter. Only alerts on detected changes.
- **Popular packages list**: `internal/analyzer/popular_packages.json` — embedded via `go:embed`. ~500 top npm packages for typosquatting detection.

## Common tasks

- **Add a new analyzer**: Implement the `Analyzer` interface in `internal/analyzer/`, add it to `All()` in `analyzer.go`, write tests.
- **Update popular packages list**: Edit `internal/analyzer/popular_packages.json`.
- **Change default thresholds**: Edit `internal/config/config.go` `Default()` function.
