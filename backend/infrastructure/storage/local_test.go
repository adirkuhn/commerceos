package storage_test

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/infrastructure/storage"
)

func TestLocalStorage_saveWritesFileAndReturnsURLUnderBaseURL(t *testing.T) {
	dir := t.TempDir()
	s := storage.NewLocalStorage(dir, "https://api.example.com/api/v1/uploads")

	url, err := s.Save(context.Background(), "tenant-1", "photo.jpg", strings.NewReader("fake-image-bytes"))

	require.NoError(t, err)
	require.True(t, strings.HasPrefix(url, "https://api.example.com/api/v1/uploads/tenant-1/"))
	require.True(t, strings.HasSuffix(url, ".jpg"), "must preserve the original extension")

	// the file must actually exist on disk under tenant-1/
	entries, err := os.ReadDir(filepath.Join(dir, "tenant-1"))
	require.NoError(t, err)
	require.Len(t, entries, 1)

	written, err := os.ReadFile(filepath.Join(dir, "tenant-1", entries[0].Name()))
	require.NoError(t, err)
	require.Equal(t, "fake-image-bytes", string(written))
}

func TestLocalStorage_saveGeneratesUniqueFilenamesForSameOriginalName(t *testing.T) {
	dir := t.TempDir()
	s := storage.NewLocalStorage(dir, "https://api.example.com/api/v1/uploads")

	url1, err := s.Save(context.Background(), "tenant-1", "photo.jpg", strings.NewReader("a"))
	require.NoError(t, err)
	url2, err := s.Save(context.Background(), "tenant-1", "photo.jpg", strings.NewReader("b"))
	require.NoError(t, err)

	require.NotEqual(t, url1, url2)
}

func TestLocalStorage_deleteRemovesTheFile(t *testing.T) {
	dir := t.TempDir()
	s := storage.NewLocalStorage(dir, "https://api.example.com/api/v1/uploads")

	url, err := s.Save(context.Background(), "tenant-1", "photo.jpg", strings.NewReader("a"))
	require.NoError(t, err)

	require.NoError(t, s.Delete(context.Background(), url))

	entries, err := os.ReadDir(filepath.Join(dir, "tenant-1"))
	require.NoError(t, err)
	require.Empty(t, entries)
}

func TestLocalStorage_getReadsBackWhatWasSaved(t *testing.T) {
	dir := t.TempDir()
	s := storage.NewLocalStorage(dir, "https://api.example.com/api/v1/uploads")

	url, err := s.Save(context.Background(), "tenant-1", "photo.jpg", strings.NewReader("fake-image-bytes"))
	require.NoError(t, err)

	rc, err := s.Get(context.Background(), url)
	require.NoError(t, err)
	defer rc.Close()

	data, err := io.ReadAll(rc)
	require.NoError(t, err)
	require.Equal(t, "fake-image-bytes", string(data))
}

func TestLocalStorage_getOfMissingFileReturnsError(t *testing.T) {
	dir := t.TempDir()
	s := storage.NewLocalStorage(dir, "https://api.example.com/api/v1/uploads")

	_, err := s.Get(context.Background(), "https://api.example.com/api/v1/uploads/tenant-1/does-not-exist.jpg")

	require.Error(t, err)
}

func TestLocalStorage_deleteOfMissingFileIsNotAnError(t *testing.T) {
	dir := t.TempDir()
	s := storage.NewLocalStorage(dir, "https://api.example.com/api/v1/uploads")

	err := s.Delete(context.Background(), "https://api.example.com/api/v1/uploads/tenant-1/does-not-exist.jpg")

	require.NoError(t, err)
}

func TestLocalStorage_rejectsPathTraversalInFilename(t *testing.T) {
	dir := t.TempDir()
	s := storage.NewLocalStorage(dir, "https://api.example.com/api/v1/uploads")

	_, err := s.Save(context.Background(), "tenant-1", "../../etc/passwd", strings.NewReader("a"))

	require.Error(t, err)
}
