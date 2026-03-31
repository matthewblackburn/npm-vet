package parser

import "strings"

// ParsedArgs holds the result of parsing npm CLI arguments.
type ParsedArgs struct {
	IsInstall bool     // true if the command is an install variant
	Packages  []string // package specifiers (e.g. "express", "@types/node@^5.0.0")
	NpmArgs   []string // original args to pass through to real npm
}

// install command aliases recognized by npm
var installCommands = map[string]bool{
	"install":  true,
	"i":        true,
	"add":      true,
	"isntall":  true,
	"ci":       true,
}

// ParseArgs parses raw npm CLI arguments to determine if this is an install
// command and extract package specifiers.
//
// Examples:
//
//	["install", "express"]                → IsInstall=true, Packages=["express"]
//	["install", "@types/node@^5.0.0"]    → IsInstall=true, Packages=["@types/node@^5.0.0"]
//	["install"]                           → IsInstall=true, Packages=[] (bare install)
//	["test"]                              → IsInstall=false
//	["run", "build"]                      → IsInstall=false
func ParseArgs(args []string) ParsedArgs {
	result := ParsedArgs{
		NpmArgs: args,
	}

	if len(args) == 0 {
		return result
	}

	// Find the command (first non-flag argument)
	cmdIdx := -1
	for i, arg := range args {
		if !strings.HasPrefix(arg, "-") {
			cmdIdx = i
			break
		}
	}

	if cmdIdx == -1 {
		return result
	}

	cmd := args[cmdIdx]
	if !installCommands[cmd] {
		return result
	}

	result.IsInstall = true

	// Extract package specifiers from remaining args.
	// Package specifiers are non-flag args after the command.
	// Scoped packages start with @ (e.g. @types/node, @babel/core@7).
	for _, arg := range args[cmdIdx+1:] {
		if isNpmFlag(arg) {
			continue
		}
		// Skip if it looks like a flag value (handled by isNpmFlag for --key=value,
		// but standalone values after flags like --registry https://... are tricky).
		// For simplicity, treat any non-flag arg as a package specifier.
		result.Packages = append(result.Packages, arg)
	}

	return result
}

// SplitPackageSpec splits a package specifier into name and version range.
//
// Examples:
//
//	"express"           → ("express", "")
//	"express@^5.0.0"    → ("express", "^5.0.0")
//	"@types/node"       → ("@types/node", "")
//	"@types/node@^20"   → ("@types/node", "^20")
func SplitPackageSpec(spec string) (name, versionRange string) {
	// Scoped packages: @scope/name or @scope/name@version
	if strings.HasPrefix(spec, "@") {
		// Find the second @ (version separator) after the scope
		rest := spec[1:] // strip leading @
		atIdx := strings.Index(rest, "@")
		if atIdx == -1 {
			return spec, ""
		}
		// Make sure the @ is after the slash (i.e., it's a version separator, not part of scope)
		slashIdx := strings.Index(rest, "/")
		if slashIdx == -1 || atIdx <= slashIdx {
			// Malformed or @ is within scope — treat whole thing as name
			return spec, ""
		}
		return spec[:atIdx+1], spec[atIdx+2:] // +1 for the leading @, +2 to skip past the @
	}

	// Unscoped: name or name@version
	atIdx := strings.Index(spec, "@")
	if atIdx == -1 {
		return spec, ""
	}
	return spec[:atIdx], spec[atIdx+1:]
}

// isNpmFlag returns true if the argument is a flag (starts with - or --).
func isNpmFlag(arg string) bool {
	return strings.HasPrefix(arg, "-")
}
