package reporter

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/matthewblackburn/npm-vet/internal/npmvet"
)

// SARIF types for GitHub code scanning integration (SARIF 2.1.0)

type sarifLog struct {
	Version string     `json:"version"`
	Schema  string     `json:"$schema"`
	Runs    []sarifRun `json:"runs"`
}

type sarifRun struct {
	Tool    sarifTool     `json:"tool"`
	Results []sarifResult `json:"results"`
}

type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}

type sarifDriver struct {
	Name           string                `json:"name"`
	Version        string                `json:"version"`
	InformationURI string                `json:"informationUri"`
	Rules          []sarifReportingDesc  `json:"rules"`
}

type sarifReportingDesc struct {
	ID               string           `json:"id"`
	ShortDescription sarifMessage     `json:"shortDescription"`
	HelpURI          string           `json:"helpUri,omitempty"`
	DefaultConfig    sarifRuleConfig  `json:"defaultConfiguration"`
}

type sarifRuleConfig struct {
	Level string `json:"level"`
}

type sarifResult struct {
	RuleID  string        `json:"ruleId"`
	Level   string        `json:"level"`
	Message sarifMessage  `json:"message"`
}

type sarifMessage struct {
	Text string `json:"text"`
}

// SARIFReport writes findings in SARIF 2.1.0 format to stdout.
func SARIFReport(findings []npmvet.Finding, version string) error {
	if findings == nil {
		findings = []npmvet.Finding{}
	}

	// Build unique rules from analyzer names
	ruleMap := map[string]bool{}
	var rules []sarifReportingDesc
	for _, f := range findings {
		if !ruleMap[f.Analyzer] {
			ruleMap[f.Analyzer] = true
			rules = append(rules, sarifReportingDesc{
				ID:               f.Analyzer,
				ShortDescription: sarifMessage{Text: f.Analyzer + " check"},
				DefaultConfig:    sarifRuleConfig{Level: sarifLevel(f.Severity)},
			})
		}
	}

	var results []sarifResult
	for _, f := range findings {
		results = append(results, sarifResult{
			RuleID:  f.Analyzer,
			Level:   sarifLevel(f.Severity),
			Message: sarifMessage{Text: fmt.Sprintf("[%s] %s: %s", f.Package, f.Title, f.Detail)},
		})
	}

	log := sarifLog{
		Version: "2.1.0",
		Schema:  "https://json.schemastore.org/sarif-2.1.0.json",
		Runs: []sarifRun{{
			Tool: sarifTool{
				Driver: sarifDriver{
					Name:           "npm-vet",
					Version:        version,
					InformationURI: "https://github.com/matthewblackburn/npm-vet",
					Rules:          rules,
				},
			},
			Results: results,
		}},
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(log); err != nil {
		return fmt.Errorf("encoding SARIF report: %w", err)
	}
	return nil
}

func sarifLevel(s npmvet.Severity) string {
	switch s {
	case npmvet.SeverityCritical:
		return "error"
	case npmvet.SeverityWarning:
		return "warning"
	default:
		return "note"
	}
}
