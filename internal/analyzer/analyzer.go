package analyzer

import (
	"github.com/matthewblackburn/npm-vet/internal/npmvet"
	"github.com/matthewblackburn/npm-vet/internal/registry"
)

// AnalyzerContext provides data for analyzers to inspect.
type AnalyzerContext struct {
	Metadata  *registry.PackageMetadata
	Downloads *registry.DownloadStats // nil if fetch failed
	Tarball   []registry.TarballFile  // nil unless analyzer needs tarball
}

// Analyzer inspects a package and returns security findings.
type Analyzer interface {
	Name() string
	NeedsTarball() bool
	Analyze(pkg npmvet.PackageSpec, ctx AnalyzerContext) ([]npmvet.Finding, error)
}

// All returns all built-in analyzers.
func All() []Analyzer {
	return []Analyzer{
		&PostinstallAnalyzer{},
		&MaintainerAnalyzer{},
		&TyposquatAnalyzer{},
		&DownloadsAnalyzer{},
		&ObfuscationAnalyzer{},
		&NetworkAnalyzer{},
	}
}
