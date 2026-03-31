package registry

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	metadataTTL  = 1 * time.Hour
	downloadsTTL = 6 * time.Hour
)

// Cache provides disk-based caching for registry responses.
type Cache struct {
	dir string
}

// NewCache creates a cache in the given directory, creating it if needed.
func NewCache(dir string) (*Cache, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("creating cache dir: %w", err)
	}
	return &Cache{dir: dir}, nil
}

// DefaultCacheDir returns the default cache directory (~/.npm-vet/cache).
func DefaultCacheDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".npm-vet", "cache")
}

// GetMetadata retrieves cached package metadata if it exists and hasn't expired.
func (c *Cache) GetMetadata(name string) (*FullPackageMetadata, bool) {
	var meta FullPackageMetadata
	if c.get("metadata", name, metadataTTL, &meta) {
		return &meta, true
	}
	return nil, false
}

// SetMetadata caches package metadata.
func (c *Cache) SetMetadata(name string, meta *FullPackageMetadata) {
	c.set("metadata", name, meta)
}

// GetDownloads retrieves cached download stats if available and fresh.
func (c *Cache) GetDownloads(name string) (*DownloadStats, bool) {
	var stats DownloadStats
	if c.get("downloads", name, downloadsTTL, &stats) {
		return &stats, true
	}
	return nil, false
}

// SetDownloads caches download stats.
func (c *Cache) SetDownloads(name string, stats *DownloadStats) {
	c.set("downloads", name, stats)
}

func (c *Cache) get(bucket, key string, ttl time.Duration, dest interface{}) bool {
	path := c.path(bucket, key)
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	if time.Since(info.ModTime()) > ttl {
		return false
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}

	return json.Unmarshal(data, dest) == nil
}

func (c *Cache) set(bucket, key string, value interface{}) {
	dir := filepath.Join(c.dir, bucket)
	os.MkdirAll(dir, 0o755)

	data, err := json.Marshal(value)
	if err != nil {
		return
	}

	path := c.path(bucket, key)
	// Write errors are non-fatal (e.g., read-only filesystem in CI)
	os.WriteFile(path, data, 0o644)
}

func (c *Cache) path(bucket, key string) string {
	// Hash the key to handle scoped packages with slashes
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(key)))
	return filepath.Join(c.dir, bucket, hash[:16]+".json")
}
