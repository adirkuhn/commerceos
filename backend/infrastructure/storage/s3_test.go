package storage

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubS3Client struct {
	putCalled    bool
	putBucket    string
	putObjectKey string
	putSize      int64
	putOpts      minio.PutObjectOptions
	putErr       error

	removeCalled bool
	removeBucket string
	removeKey    string
	removeErr    error

	getCalled bool
	getBucket string
	getKey    string
	getErr    error
}

func (s *stubS3Client) PutObject(_ context.Context, bucketName, objectName string, reader io.Reader, objectSize int64, opts minio.PutObjectOptions) (minio.UploadInfo, error) {
	s.putCalled = true
	s.putBucket = bucketName
	s.putObjectKey = objectName
	s.putSize = objectSize
	s.putOpts = opts
	if reader != nil {
		_, _ = io.Copy(io.Discard, reader)
	}
	if s.putErr != nil {
		return minio.UploadInfo{}, s.putErr
	}
	return minio.UploadInfo{Bucket: bucketName, Key: objectName}, nil
}

func (s *stubS3Client) RemoveObject(_ context.Context, bucketName, objectName string, _ minio.RemoveObjectOptions) error {
	s.removeCalled = true
	s.removeBucket = bucketName
	s.removeKey = objectName
	return s.removeErr
}

// GetObject only records the call and returns a zero-value *minio.Object --
// minio.Object's fields are all unexported (constructed internally by the
// real SDK client via a network round-trip), so a stub can't fake a working
// readable body. That means this stub can only verify Get's routing logic
// (does it ask the client for the right bucket/key), not the byte stream --
// end-to-end read-back is exercised live against the real MinIO container
// this repo already runs in docker-compose, not here.
func (s *stubS3Client) GetObject(_ context.Context, bucketName, objectName string, _ minio.GetObjectOptions) (*minio.Object, error) {
	s.getCalled = true
	s.getBucket = bucketName
	s.getKey = objectName
	if s.getErr != nil {
		return nil, s.getErr
	}
	return &minio.Object{}, nil
}

func newS3Storage(client s3PutRemover, bucket, publicBaseURL string) *S3Storage {
	return &S3Storage{client: client, bucket: bucket, publicBaseURL: strings.TrimSuffix(publicBaseURL, "/")}
}

func TestS3Storage_Save_uploadsAndReturnsPublicURL(t *testing.T) {
	client := &stubS3Client{}
	s := newS3Storage(client, "product-media", "https://media.example.com")

	url, err := s.Save(context.Background(), "tenant-1", "photo.jpg", bytes.NewReader([]byte("bytes")))

	require.NoError(t, err)
	assert.True(t, client.putCalled)
	assert.Equal(t, "product-media", client.putBucket)
	assert.True(t, strings.HasPrefix(client.putObjectKey, "tenant-1/"))
	assert.True(t, strings.HasSuffix(client.putObjectKey, ".jpg"))
	assert.Equal(t, "image/jpeg", client.putOpts.ContentType)
	assert.Equal(t, "https://media.example.com/"+client.putObjectKey, url)
}

// TestS3Storage_Save_passesExactSizeForASeekableReader is the production
// fix: every real caller (application/pms.UploadMediaUseCase,
// application/procurement's document/invoice uploads) passes
// bytes.NewReader(data) -- a fully-buffered, seekable upload whose exact
// size is always known up front. Passing that real size instead of -1
// avoids minio-go's chunked/streaming SigV4 signing path entirely (see
// knownSize's doc comment for why that matters).
func TestS3Storage_Save_passesExactSizeForASeekableReader(t *testing.T) {
	client := &stubS3Client{}
	s := newS3Storage(client, "product-media", "https://media.example.com")

	_, err := s.Save(context.Background(), "tenant-1", "invoice.pdf", bytes.NewReader([]byte("hello world")))

	require.NoError(t, err)
	assert.Equal(t, int64(11), client.putSize)
}

// TestS3Storage_Save_fallsBackToUnknownSizeForANonSeekableReader proves
// knownSize degrades safely (-1, matching the old always-streaming
// behavior) rather than erroring when given a reader with no way to
// determine its length up front -- Save's io.Reader parameter doesn't
// guarantee seekability, even though every current caller happens to
// provide it.
func TestS3Storage_Save_fallsBackToUnknownSizeForANonSeekableReader(t *testing.T) {
	client := &stubS3Client{}
	s := newS3Storage(client, "product-media", "https://media.example.com")

	_, err := s.Save(context.Background(), "tenant-1", "photo.jpg", readOnlyReader{bytes.NewBufferString("bytes")})

	require.NoError(t, err)
	assert.Equal(t, int64(-1), client.putSize)
}

// readOnlyReader strips any Seek method a wrapped reader might have --
// used to construct a genuinely non-seekable io.Reader for tests, since
// most of Go's stdlib readers (bytes.Reader, strings.Reader) implement
// io.Seeker even when constructed from an in-memory buffer.
type readOnlyReader struct{ r io.Reader }

func (r readOnlyReader) Read(p []byte) (int, error) { return r.r.Read(p) }

func TestS3Storage_Save_generatesUniqueKeysPerCall(t *testing.T) {
	client := &stubS3Client{}
	s := newS3Storage(client, "product-media", "https://media.example.com")

	url1, err := s.Save(context.Background(), "tenant-1", "photo.jpg", bytes.NewBufferString("a"))
	require.NoError(t, err)
	url2, err := s.Save(context.Background(), "tenant-1", "photo.jpg", bytes.NewBufferString("b"))
	require.NoError(t, err)

	assert.NotEqual(t, url1, url2)
}

func TestS3Storage_Save_infersContentTypeFromExtension(t *testing.T) {
	client := &stubS3Client{}
	s := newS3Storage(client, "product-media", "https://media.example.com")

	_, err := s.Save(context.Background(), "tenant-1", "photo.png", bytes.NewBufferString("bytes"))

	require.NoError(t, err)
	assert.Equal(t, "image/png", client.putOpts.ContentType)
}

func TestS3Storage_Save_fallsBackToOctetStreamForUnknownExtension(t *testing.T) {
	client := &stubS3Client{}
	s := newS3Storage(client, "product-media", "https://media.example.com")

	_, err := s.Save(context.Background(), "tenant-1", "photo.unknownext", bytes.NewBufferString("bytes"))

	require.NoError(t, err)
	assert.Equal(t, "application/octet-stream", client.putOpts.ContentType)
}

func TestS3Storage_Save_rejectsFilenameWithPathSeparators(t *testing.T) {
	client := &stubS3Client{}
	s := newS3Storage(client, "product-media", "https://media.example.com")

	_, err := s.Save(context.Background(), "tenant-1", "../../etc/passwd", bytes.NewBufferString("bytes"))

	require.Error(t, err)
	assert.False(t, client.putCalled)
}

func TestS3Storage_Save_wrapsClientError(t *testing.T) {
	client := &stubS3Client{putErr: errors.New("network down")}
	s := newS3Storage(client, "product-media", "https://media.example.com")

	_, err := s.Save(context.Background(), "tenant-1", "photo.jpg", bytes.NewBufferString("bytes"))

	require.Error(t, err)
	assert.ErrorContains(t, err, "network down")
}

func TestS3Storage_Delete_removesObjectByKey(t *testing.T) {
	client := &stubS3Client{}
	s := newS3Storage(client, "product-media", "https://media.example.com")

	err := s.Delete(context.Background(), "https://media.example.com/tenant-1/abc.jpg")

	require.NoError(t, err)
	assert.True(t, client.removeCalled)
	assert.Equal(t, "product-media", client.removeBucket)
	assert.Equal(t, "tenant-1/abc.jpg", client.removeKey)
}

func TestS3Storage_Delete_wrapsClientError(t *testing.T) {
	client := &stubS3Client{removeErr: errors.New("bucket unreachable")}
	s := newS3Storage(client, "product-media", "https://media.example.com")

	err := s.Delete(context.Background(), "https://media.example.com/tenant-1/abc.jpg")

	require.Error(t, err)
	assert.ErrorContains(t, err, "bucket unreachable")
}

func TestS3Storage_Get_requestsObjectByKey(t *testing.T) {
	client := &stubS3Client{}
	s := newS3Storage(client, "product-media", "https://media.example.com")

	rc, err := s.Get(context.Background(), "https://media.example.com/tenant-1/abc.jpg")

	require.NoError(t, err)
	require.NotNil(t, rc)
	assert.True(t, client.getCalled)
	assert.Equal(t, "product-media", client.getBucket)
	assert.Equal(t, "tenant-1/abc.jpg", client.getKey)
}

func TestS3Storage_Get_wrapsClientError(t *testing.T) {
	client := &stubS3Client{getErr: errors.New("object not found")}
	s := newS3Storage(client, "product-media", "https://media.example.com")

	_, err := s.Get(context.Background(), "https://media.example.com/tenant-1/abc.jpg")

	require.Error(t, err)
	assert.ErrorContains(t, err, "object not found")
}

func TestNewS3Storage_trimsTrailingSlashFromPublicBaseURL(t *testing.T) {
	client := &stubS3Client{}
	s := NewS3Storage(nil, "product-media", "https://media.example.com/")
	s.client = client

	url, err := s.Save(context.Background(), "tenant-1", "photo.jpg", bytes.NewBufferString("bytes"))

	require.NoError(t, err)
	assert.False(t, strings.Contains(url, "//tenant-1"))
}
