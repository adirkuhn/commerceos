// Package storage adapts the domain's pms.FileStorage port to a real
// backend. LocalStorage saves to the local filesystem — swapping to S3 or a
// CDN later means adding a new adapter here, never touching domain or
// application code.
package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/adirkuhn/commerceos/internal/idgen"
)

type LocalStorage struct {
	baseDir string
	baseURL string
}

func NewLocalStorage(baseDir, baseURL string) *LocalStorage {
	return &LocalStorage{baseDir: baseDir, baseURL: strings.TrimSuffix(baseURL, "/")}
}

// Save writes data under baseDir/tenantID/<generated-id><ext>, where ext is
// taken from filename. filename is never used to build the path beyond its
// extension — filepath.Base is compared against the original to reject any
// path separators (defense in depth; a bare extension can't itself escape
// baseDir, but this makes that safety explicit rather than incidental).
func (s *LocalStorage) Save(_ context.Context, tenantID, filename string, data io.Reader) (string, error) {
	if filepath.Base(filename) != filename {
		return "", fmt.Errorf("localStorage.Save: invalid filename %q", filename)
	}

	dir := filepath.Join(s.baseDir, tenantID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("localStorage.Save: %w", err)
	}

	name := idgen.NewV4() + filepath.Ext(filename)
	f, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		return "", fmt.Errorf("localStorage.Save: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, data); err != nil {
		return "", fmt.Errorf("localStorage.Save: %w", err)
	}

	return fmt.Sprintf("%s/%s/%s", s.baseURL, tenantID, name), nil
}

// Delete removes the file at url. Deleting a file that's already gone is
// not an error — the end state (file absent) is what the caller wants.
func (s *LocalStorage) Delete(_ context.Context, url string) error {
	rel := strings.TrimPrefix(url, s.baseURL+"/")
	err := os.Remove(filepath.Join(s.baseDir, rel))
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("localStorage.Delete: %w", err)
	}
	return nil
}

// Get re-opens a previously-saved file for reading. The caller owns the
// returned ReadCloser and must Close it.
func (s *LocalStorage) Get(_ context.Context, url string) (io.ReadCloser, error) {
	rel := strings.TrimPrefix(url, s.baseURL+"/")
	f, err := os.Open(filepath.Join(s.baseDir, rel))
	if err != nil {
		return nil, fmt.Errorf("localStorage.Get: %w", err)
	}
	return f, nil
}
