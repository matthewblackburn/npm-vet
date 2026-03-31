package reporter

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/matthewblackburn/npm-vet/internal/npmvet"
)

// JSONReport writes findings as a JSON array to stdout.
func JSONReport(findings []npmvet.Finding) error {
	if findings == nil {
		findings = []npmvet.Finding{}
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(findings); err != nil {
		return fmt.Errorf("encoding JSON report: %w", err)
	}
	return nil
}
