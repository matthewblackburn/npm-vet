package registry

import "time"

// PackageMetadata represents the metadata returned by the npm registry for a package version.
type PackageMetadata struct {
	Name        string                `json:"name"`
	Version     string                `json:"version"`
	Description string                `json:"description"`
	Maintainers []Maintainer          `json:"maintainers"`
	Scripts     map[string]string     `json:"scripts"`
	Dist        Dist                  `json:"dist"`
	Time        map[string]string     `json:"time"` // version → ISO timestamp
}

// Maintainer represents a package maintainer.
type Maintainer struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// Dist contains distribution metadata for a package version.
type Dist struct {
	Tarball   string `json:"tarball"`
	Shasum    string `json:"shasum"`
	Integrity string `json:"integrity"`
}

// DownloadStats represents weekly download statistics for a package.
type DownloadStats struct {
	Downloads int    `json:"downloads"`
	Start     string `json:"start"`
	End       string `json:"end"`
	Package   string `json:"package"`
}

// TarballFile represents a single file extracted from a package tarball.
type TarballFile struct {
	Path    string
	Content string
	Size    int64
}

// FullPackageMetadata is the full registry response for a package (all versions).
type FullPackageMetadata struct {
	Name     string                        `json:"name"`
	DistTags map[string]string             `json:"dist-tags"`
	Time     map[string]string             `json:"time"`
	Versions map[string]VersionMetadata    `json:"versions"`
}

// VersionMetadata is the per-version metadata within the full package document.
type VersionMetadata struct {
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Description string            `json:"description"`
	Maintainers []Maintainer      `json:"maintainers"`
	Scripts     map[string]string `json:"scripts"`
	Dist        Dist              `json:"dist"`
}

// PublishTime returns the parsed publish time for the given version, or zero time if unavailable.
func (m *PackageMetadata) PublishTime(version string) time.Time {
	if ts, ok := m.Time[version]; ok {
		t, err := time.Parse(time.RFC3339, ts)
		if err != nil {
			// Try alternate format
			t, err = time.Parse("2006-01-02T15:04:05.000Z", ts)
			if err != nil {
				return time.Time{}
			}
		}
		return t
	}
	return time.Time{}
}
