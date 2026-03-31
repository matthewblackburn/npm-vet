package registry

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

const (
	defaultRegistryURL = "https://registry.npmjs.org"
	downloadsURL       = "https://api.npmjs.org/downloads/point/last-week"
	requestTimeout     = 10 * time.Second
	maxTarballSize     = 5 * 1024 * 1024 // 5MB
)

// Client fetches package metadata, download stats, and tarballs from the npm registry.
type Client struct {
	httpClient  *http.Client
	registryURL string
}

// NewClient creates a new registry client with the configured timeout.
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: requestTimeout,
		},
		registryURL: defaultRegistryURL,
	}
}

// GetPackageMetadata fetches metadata for a specific package.
// If version is empty, fetches the full document (all versions).
func (c *Client) GetPackageMetadata(ctx context.Context, name string) (*FullPackageMetadata, error) {
	url := fmt.Sprintf("%s/%s", c.registryURL, name)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching %s: %w", name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("package %q not found", name)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry returned %d for %s", resp.StatusCode, name)
	}

	var meta FullPackageMetadata
	if err := json.NewDecoder(resp.Body).Decode(&meta); err != nil {
		return nil, fmt.Errorf("decoding metadata for %s: %w", name, err)
	}

	return &meta, nil
}

// GetVersionMetadata fetches metadata for a specific version of a package.
// It returns a PackageMetadata that includes the time map from the full document.
func (c *Client) GetVersionMetadata(ctx context.Context, name, version string) (*PackageMetadata, error) {
	full, err := c.GetPackageMetadata(ctx, name)
	if err != nil {
		return nil, err
	}

	// Resolve "latest" or other dist-tags
	if resolved, ok := full.DistTags[version]; ok {
		version = resolved
	}

	// If version is empty, use latest
	if version == "" {
		latest, ok := full.DistTags["latest"]
		if !ok {
			return nil, fmt.Errorf("no latest version found for %s", name)
		}
		version = latest
	}

	vm, ok := full.Versions[version]
	if !ok {
		return nil, fmt.Errorf("version %s not found for %s", version, name)
	}

	return &PackageMetadata{
		Name:        vm.Name,
		Version:     vm.Version,
		Description: vm.Description,
		Maintainers: vm.Maintainers,
		Scripts:     vm.Scripts,
		Dist:        vm.Dist,
		Time:        full.Time,
	}, nil
}

// ResolveVersion resolves a version range or dist-tag to a concrete version.
// For now, this resolves "latest" and exact versions. Semver range resolution
// is handled by using the lockfile when available.
func (c *Client) ResolveVersion(ctx context.Context, name, versionRange string) (string, error) {
	full, err := c.GetPackageMetadata(ctx, name)
	if err != nil {
		return "", err
	}

	// Check dist-tags first
	if versionRange == "" || versionRange == "latest" {
		if v, ok := full.DistTags["latest"]; ok {
			return v, nil
		}
		return "", fmt.Errorf("no latest tag for %s", name)
	}

	// Check if it's a dist-tag
	if v, ok := full.DistTags[versionRange]; ok {
		return v, nil
	}

	// Check if it's an exact version
	if _, ok := full.Versions[versionRange]; ok {
		return versionRange, nil
	}

	// For semver ranges, fall back to latest (proper semver resolution would
	// require a semver library, but lockfile fast-path handles most cases)
	if v, ok := full.DistTags["latest"]; ok {
		return v, nil
	}

	return "", fmt.Errorf("cannot resolve version %q for %s", versionRange, name)
}

// GetDownloadStats fetches weekly download statistics for a package.
func (c *Client) GetDownloadStats(ctx context.Context, name string) (*DownloadStats, error) {
	url := fmt.Sprintf("%s/%s", downloadsURL, name)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching downloads for %s: %w", name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("downloads API returned %d for %s", resp.StatusCode, name)
	}

	var stats DownloadStats
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, fmt.Errorf("decoding downloads for %s: %w", name, err)
	}

	return &stats, nil
}

// DownloadTarball downloads and extracts a package tarball, returning its JS/JSON files.
// Returns an error if the tarball exceeds maxTarballSize.
func (c *Client) DownloadTarball(ctx context.Context, tarballURL string) ([]TarballFile, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", tarballURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating tarball request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("downloading tarball: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tarball download returned %d", resp.StatusCode)
	}

	// Check Content-Length if available
	if resp.ContentLength > int64(maxTarballSize) {
		return nil, fmt.Errorf("tarball too large (%d bytes, max %d)", resp.ContentLength, maxTarballSize)
	}

	// Limit reader to prevent downloading more than max
	limitedReader := io.LimitReader(resp.Body, int64(maxTarballSize)+1)

	gz, err := gzip.NewReader(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("decompressing tarball: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	var files []TarballFile
	var totalSize int64

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("reading tarball: %w", err)
		}

		if header.Typeflag != tar.TypeReg {
			continue
		}

		totalSize += header.Size
		if totalSize > int64(maxTarballSize) {
			return nil, fmt.Errorf("tarball contents too large (exceeds %d bytes)", maxTarballSize)
		}

		// Only extract JS, CJS, MJS, and JSON files
		ext := strings.ToLower(filepath.Ext(header.Name))
		if ext != ".js" && ext != ".cjs" && ext != ".mjs" && ext != ".json" {
			continue
		}

		content, err := io.ReadAll(io.LimitReader(tr, header.Size))
		if err != nil {
			return nil, fmt.Errorf("reading file %s: %w", header.Name, err)
		}

		files = append(files, TarballFile{
			Path:    header.Name,
			Content: string(content),
			Size:    header.Size,
		})
	}

	return files, nil
}
