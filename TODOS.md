# TODOs

## pnpm/yarn support

**What:** Parse `pnpm-lock.yaml` and `yarn.lock` in addition to `package-lock.json`.

**Why:** Many teams use pnpm or yarn as their package manager. Without lockfile support for these, the tool falls back to registry resolution (slower, less precise). This is the most requested feature gap for adoption in mixed-toolchain teams.

**Context:** The lockfile fast-path in `internal/parser/packagejson.go` currently only handles npm's `package-lock.json` (v2/v3 format). pnpm uses YAML-based lockfiles with a different structure (`pnpm-lock.yaml`). Yarn v1 uses a custom format (`yarn.lock`), and Yarn v3+ uses YAML. Each needs a separate parser. The `ReadLockfile` function should detect which lockfile exists and dispatch to the right parser.

**Depends on:** Stable npm lockfile support (done).

## GitHub Actions marketplace action

**What:** Publish a GitHub Action that wraps `npm-vet audit --sarif` and uploads results to GitHub Advanced Security code scanning.

**Why:** One-line CI integration (`uses: matthewblackburn/npm-vet-action@v1`) would dramatically lower the adoption barrier. Currently users have to download the binary and invoke it manually in their CI config.

**Context:** The action would: (1) download the pre-built binary for the runner OS/arch, (2) run `npm-vet audit --sarif > results.sarif`, (3) upload via `github/codeql-action/upload-sarif`. The SARIF reporter (`internal/reporter/sarif.go`) already outputs valid SARIF 2.1.0. The binary cross-compilation targets are in the Makefile.

**Depends on:** Stable SARIF output (done), GitHub release automation (not yet done).

## --maintainer-baseline flag

**What:** Allow providing a known-good maintainer snapshot file via `--maintainer-baseline=path/to/baseline.json` for CI environments.

**Why:** The maintainer analyzer is currently a no-op on first encounter (by design — no cache to compare against). In ephemeral CI runners, every run is a "first encounter." A baseline file would let teams track maintainer changes across CI runs by committing the baseline to their repo.

**Context:** The maintainer cache format is already defined in `internal/analyzer/maintainer.go` (`maintainerCache` struct). The baseline would use the same format. Implementation: if `--maintainer-baseline` is set, load it as the "previous" state instead of the disk cache. Optionally support `--update-baseline` to write the current state back to the file after analysis. Teams would commit this file and review diffs in PRs.

**Depends on:** Stable maintainer analyzer (done).
