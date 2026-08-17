package storage

import (
	"context"
	"fmt"
	"io"
	"mime"
	"path/filepath"
	"strings"

	"github.com/adirkuhn/commerceos/internal/idgen"
	"github.com/minio/minio-go/v7"
)

// s3PutRemover is the narrow slice of *minio.Client this adapter needs —
// defined at point of use so tests can stub it without a real MinIO/S3
// server.
type s3PutRemover interface {
	PutObject(ctx context.Context, bucketName, objectName string, reader io.Reader, objectSize int64, opts minio.PutObjectOptions) (minio.UploadInfo, error)
	RemoveObject(ctx context.Context, bucketName, objectName string, opts minio.RemoveObjectOptions) error
	GetObject(ctx context.Context, bucketName, objectName string, opts minio.GetObjectOptions) (*minio.Object, error)
}

// S3Storage adapts the domain's pms.FileStorage port to an S3-compatible
// object store (MinIO self-hosted, or real AWS S3 — same protocol).
type S3Storage struct {
	client        s3PutRemover
	bucket        string
	publicBaseURL string
}

func NewS3Storage(client *minio.Client, bucket, publicBaseURL string) *S3Storage {
	return &S3Storage{client: client, bucket: bucket, publicBaseURL: strings.TrimSuffix(publicBaseURL, "/")}
}

// Save uploads data under tenantID/<generated-id><ext>, where ext is taken
// from filename. filename is never used to build the key beyond its
// extension — same defense-in-depth check as LocalStorage.
func (s *S3Storage) Save(ctx context.Context, tenantID, filename string, data io.Reader) (string, error) {
	if filepath.Base(filename) != filename {
		return "", fmt.Errorf("s3Storage.Save: invalid filename %q", filename)
	}

	ext := filepath.Ext(filename)
	key := fmt.Sprintf("%s/%s%s", tenantID, idgen.NewV4(), ext)

	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	size, err := knownSize(data)
	if err != nil {
		return "", fmt.Errorf("s3Storage.Save: %w", err)
	}
	if _, err := s.client.PutObject(ctx, s.bucket, key, data, size, minio.PutObjectOptions{ContentType: contentType}); err != nil {
		return "", fmt.Errorf("s3Storage.Save: %w", err)
	}

	return fmt.Sprintf("%s/%s", s.publicBaseURL, key), nil
}

// knownSize returns data's exact byte length when it's seekable -- every
// caller in this codebase passes bytes.NewReader(data), a fully-buffered
// upload that's always seekable -- or -1 (unknown/streaming) otherwise.
// This matters beyond the response's Content-Length: minio-go signs a
// PutObject with a known size using a single standard SigV4 signature, but
// an unknown size (-1) forces STREAMING-AWS4-HMAC-SHA256-PAYLOAD chunked
// signing instead -- a request shape far more likely to break if anything
// between here and the S3-compatible server (a reverse proxy, in
// particular) re-chunks or buffers the body. Confirmed in production:
// PutObject failed with "SignatureDoesNotMatch" while mc (minio's own
// CLI, small non-streaming requests) worked fine through the same path
// with the same credentials.
func knownSize(data io.Reader) (int64, error) {
	seeker, ok := data.(io.Seeker)
	if !ok {
		return -1, nil
	}
	n, err := seeker.Seek(0, io.SeekEnd)
	if err != nil {
		return 0, err
	}
	if _, err := seeker.Seek(0, io.SeekStart); err != nil {
		return 0, err
	}
	return n, nil
}

// Delete removes the object at url. S3-compatible DELETE is idempotent —
// removing an already-gone key is not an error, so no extra handling is
// needed here (unlike LocalStorage's os.Remove, which returns ENOENT).
func (s *S3Storage) Delete(ctx context.Context, url string) error {
	key := strings.TrimPrefix(url, s.publicBaseURL+"/")
	if err := s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("s3Storage.Delete: %w", err)
	}
	return nil
}

// Get re-fetches a previously-saved object. The caller owns the returned
// ReadCloser and must Close it.
func (s *S3Storage) Get(ctx context.Context, url string) (io.ReadCloser, error) {
	key := strings.TrimPrefix(url, s.publicBaseURL+"/")
	obj, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("s3Storage.Get: %w", err)
	}
	return obj, nil
}
