package analyzer

import (
	"fmt"
	"time"

	"github.com/matthewblackburn/npm-vet/internal/npmvet"
)

// DownloadsAnalyzer flags packages with low download counts, especially new ones.
type DownloadsAnalyzer struct {
	MinWeeklyDownloads int // 0 means use default (100)
}

func (a *DownloadsAnalyzer) Name() string      { return "downloads" }
func (a *DownloadsAnalyzer) NeedsTarball() bool { return false }

func (a *DownloadsAnalyzer) minDownloads() int {
	if a.MinWeeklyDownloads > 0 {
		return a.MinWeeklyDownloads
	}
	return 100
}

func (a *DownloadsAnalyzer) Analyze(pkg npmvet.PackageSpec, ctx AnalyzerContext) ([]npmvet.Finding, error) {
	if ctx.Downloads == nil {
		// Can't assess without download data — skip gracefully
		return nil, nil
	}

	var findings []npmvet.Finding
	downloads := ctx.Downloads.Downloads
	minDl := a.minDownloads()

	if downloads >= 1000 {
		// Well-established package — no concern
		return nil, nil
	}

	// Determine package age from metadata
	isNew := false
	if ctx.Metadata != nil && pkg.Version != "" {
		publishTime := ctx.Metadata.PublishTime(pkg.Version)
		if !publishTime.IsZero() && time.Since(publishTime) < 7*24*time.Hour {
			isNew = true
		}
	}

	if downloads < minDl && isNew {
		findings = append(findings, npmvet.Finding{
			Analyzer: a.Name(),
			Package:  pkg.Name,
			Severity: npmvet.SeverityCritical,
			Title:    "Very new package with low downloads",
			Detail:   fmt.Sprintf("%d weekly downloads, published within the last 7 days", downloads),
		})
	} else if downloads < minDl {
		findings = append(findings, npmvet.Finding{
			Analyzer: a.Name(),
			Package:  pkg.Name,
			Severity: npmvet.SeverityWarning,
			Title:    "Low weekly downloads",
			Detail:   fmt.Sprintf("%d weekly downloads (threshold: %d)", downloads, minDl),
		})
	} else {
		// Between minDl and 1000 — info level
		findings = append(findings, npmvet.Finding{
			Analyzer: a.Name(),
			Package:  pkg.Name,
			Severity: npmvet.SeverityInfo,
			Title:    "Moderate weekly downloads",
			Detail:   fmt.Sprintf("%d weekly downloads", downloads),
		})
	}

	return findings, nil
}
