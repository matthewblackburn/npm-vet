package pipeline

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/matthewblackburn/npm-vet/internal/analyzer"
	"github.com/matthewblackburn/npm-vet/internal/npmvet"
	"github.com/matthewblackburn/npm-vet/internal/registry"
)

const globalTimeout = 60 * time.Second

// Config controls pipeline behavior.
type Config struct {
	Analyzers  []analyzer.Analyzer
	Client     *registry.Client
	Allowlist  []string // package name patterns to skip (supports trailing *)
}

// Result holds the output of a pipeline run.
type Result struct {
	Findings []npmvet.Finding
	Warnings []string // non-fatal issues (e.g., fetch failures)
}

// Run vets the given packages through all analyzers concurrently.
func Run(ctx context.Context, packages []npmvet.PackageSpec, cfg Config) (*Result, error) {
	ctx, cancel := context.WithTimeout(ctx, globalTimeout)
	defer cancel()

	// Filter allowlisted packages
	var toVet []npmvet.PackageSpec
	for _, pkg := range packages {
		if !isAllowlisted(pkg.Name, cfg.Allowlist) {
			toVet = append(toVet, pkg)
		}
	}

	if len(toVet) == 0 {
		return &Result{}, nil
	}

	// Determine if any analyzer needs tarballs
	needsTarball := false
	for _, a := range cfg.Analyzers {
		if a.NeedsTarball() {
			needsTarball = true
			break
		}
	}

	// Fetch metadata + downloads for all packages in parallel
	type pkgData struct {
		spec      npmvet.PackageSpec
		metadata  *registry.PackageMetadata
		downloads *registry.DownloadStats
		tarball   []registry.TarballFile
	}

	var (
		mu       sync.Mutex
		result   Result
		dataMap  = make(map[string]*pkgData)
	)

	// Phase 1: Parallel fetch of metadata and downloads
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(10) // max concurrent HTTP requests

	for _, pkg := range toVet {
		pkg := pkg
		dataMap[pkg.Name] = &pkgData{spec: pkg}

		// Fetch metadata
		g.Go(func() error {
			version := pkg.Version
			if version == "" {
				version = pkg.VersionRange
			}
			meta, err := cfg.Client.GetVersionMetadata(gctx, pkg.Name, version)
			if err != nil {
				mu.Lock()
				result.Warnings = append(result.Warnings, fmt.Sprintf("metadata fetch failed for %s: %v", pkg.Name, err))
				mu.Unlock()
				return nil // graceful degradation
			}
			mu.Lock()
			dataMap[pkg.Name].metadata = meta
			// Update resolved version
			dataMap[pkg.Name].spec.Version = meta.Version
			mu.Unlock()
			return nil
		})

		// Fetch download stats
		g.Go(func() error {
			stats, err := cfg.Client.GetDownloadStats(gctx, pkg.Name)
			if err != nil {
				mu.Lock()
				result.Warnings = append(result.Warnings, fmt.Sprintf("download stats unavailable for %s: %v", pkg.Name, err))
				mu.Unlock()
				return nil
			}
			mu.Lock()
			dataMap[pkg.Name].downloads = stats
			mu.Unlock()
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("fetching package data: %w", err)
	}

	// Phase 2: Download tarballs (only if needed)
	if needsTarball {
		g2, gctx2 := errgroup.WithContext(ctx)
		g2.SetLimit(5)

		for _, pd := range dataMap {
			pd := pd
			if pd.metadata == nil || pd.metadata.Dist.Tarball == "" {
				continue
			}
			g2.Go(func() error {
				files, err := cfg.Client.DownloadTarball(gctx2, pd.metadata.Dist.Tarball)
				if err != nil {
					mu.Lock()
					result.Warnings = append(result.Warnings, fmt.Sprintf("tarball download skipped for %s: %v", pd.spec.Name, err))
					mu.Unlock()
					return nil
				}
				mu.Lock()
				pd.tarball = files
				mu.Unlock()
				return nil
			})
		}

		if err := g2.Wait(); err != nil {
			return nil, fmt.Errorf("downloading tarballs: %w", err)
		}
	}

	// Phase 3: Run analyzers concurrently per package
	g3, _ := errgroup.WithContext(ctx)
	g3.SetLimit(20)

	for _, pd := range dataMap {
		for _, a := range cfg.Analyzers {
			pd := pd
			a := a

			// Skip tarball analyzers if no tarball available
			if a.NeedsTarball() && pd.tarball == nil {
				continue
			}

			g3.Go(func() error {
				actx := analyzer.AnalyzerContext{
					Metadata:  pd.metadata,
					Downloads: pd.downloads,
					Tarball:   pd.tarball,
				}

				findings, err := a.Analyze(pd.spec, actx)
				if err != nil {
					mu.Lock()
					result.Warnings = append(result.Warnings, fmt.Sprintf("%s analyzer error for %s: %v", a.Name(), pd.spec.Name, err))
					mu.Unlock()
					return nil
				}

				if len(findings) > 0 {
					mu.Lock()
					result.Findings = append(result.Findings, findings...)
					mu.Unlock()
				}
				return nil
			})
		}
	}

	if err := g3.Wait(); err != nil {
		return nil, fmt.Errorf("running analyzers: %w", err)
	}

	// Print warnings to stderr
	for _, w := range result.Warnings {
		fmt.Fprintf(os.Stderr, "npm-vet: warning: %s\n", w)
	}

	return &result, nil
}

// isAllowlisted checks if a package name matches any allowlist pattern.
// Supports trailing wildcard: "@types/*" matches "@types/node".
func isAllowlisted(name string, patterns []string) bool {
	for _, pattern := range patterns {
		if pattern == name {
			return true
		}
		// Trailing wildcard
		if len(pattern) > 1 && pattern[len(pattern)-1] == '*' {
			prefix := pattern[:len(pattern)-1]
			if len(name) >= len(prefix) && name[:len(prefix)] == prefix {
				return true
			}
		}
	}
	return false
}
