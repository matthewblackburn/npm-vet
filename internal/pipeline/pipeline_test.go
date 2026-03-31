package pipeline

import "testing"

func TestIsAllowlisted(t *testing.T) {
	patterns := []string{"safe-pkg", "@types/*", "lodash"}

	tests := []struct {
		name string
		want bool
	}{
		{"safe-pkg", true},
		{"unsafe-pkg", false},
		{"@types/node", true},
		{"@types/react", true},
		{"@typos/node", false},
		{"lodash", true},
		{"lodash-es", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isAllowlisted(tt.name, patterns)
			if got != tt.want {
				t.Errorf("isAllowlisted(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}
