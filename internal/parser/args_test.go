package parser

import (
	"testing"
)

func TestParseArgs_InstallCommands(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		isInstall bool
		packages  []string
	}{
		{"bare install", []string{"install"}, true, nil},
		{"install alias i", []string{"i"}, true, nil},
		{"install alias add", []string{"add"}, true, nil},
		{"install alias isntall", []string{"isntall"}, true, nil},
		{"ci", []string{"ci"}, true, nil},

		{"single package", []string{"install", "express"}, true, []string{"express"}},
		{"multiple packages", []string{"install", "express", "lodash"}, true, []string{"express", "lodash"}},
		{"versioned package", []string{"install", "express@^5.0.0"}, true, []string{"express@^5.0.0"}},

		// Scoped packages
		{"scoped package", []string{"install", "@types/node"}, true, []string{"@types/node"}},
		{"scoped versioned", []string{"install", "@babel/core@7"}, true, []string{"@babel/core@7"}},
		{"scoped semver", []string{"install", "@types/node@^20.0.0"}, true, []string{"@types/node@^20.0.0"}},
		{"multiple scoped", []string{"install", "@types/node", "@babel/core@7"}, true, []string{"@types/node", "@babel/core@7"}},

		// With flags
		{"with save-dev", []string{"install", "--save-dev", "express"}, true, []string{"express"}},
		{"with -D", []string{"install", "-D", "express"}, true, []string{"express"}},
		{"flag after packages", []string{"install", "express", "--save-dev"}, true, []string{"express"}},
		{"global install", []string{"install", "--global", "typescript"}, true, []string{"typescript"}},

		// Non-install commands
		{"test", []string{"test"}, false, nil},
		{"run build", []string{"run", "build"}, false, nil},
		{"publish", []string{"publish"}, false, nil},
		{"empty args", []string{}, false, nil},
		{"only flags", []string{"--version"}, false, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseArgs(tt.args)
			if result.IsInstall != tt.isInstall {
				t.Errorf("IsInstall = %v, want %v", result.IsInstall, tt.isInstall)
			}
			if len(result.Packages) != len(tt.packages) {
				t.Errorf("Packages = %v, want %v", result.Packages, tt.packages)
				return
			}
			for i, pkg := range result.Packages {
				if pkg != tt.packages[i] {
					t.Errorf("Packages[%d] = %q, want %q", i, pkg, tt.packages[i])
				}
			}
		})
	}
}

func TestSplitPackageSpec(t *testing.T) {
	tests := []struct {
		spec         string
		wantName     string
		wantVersion  string
	}{
		{"express", "express", ""},
		{"express@^5.0.0", "express", "^5.0.0"},
		{"express@latest", "express", "latest"},
		{"@types/node", "@types/node", ""},
		{"@types/node@^20", "@types/node", "^20"},
		{"@babel/core@7.24.0", "@babel/core", "7.24.0"},
		{"@scope/pkg@~1.0.0", "@scope/pkg", "~1.0.0"},
		{"lodash@4.17.21", "lodash", "4.17.21"},
	}

	for _, tt := range tests {
		t.Run(tt.spec, func(t *testing.T) {
			name, version := SplitPackageSpec(tt.spec)
			if name != tt.wantName {
				t.Errorf("name = %q, want %q", name, tt.wantName)
			}
			if version != tt.wantVersion {
				t.Errorf("version = %q, want %q", version, tt.wantVersion)
			}
		})
	}
}
