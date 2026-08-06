package pms

import (
	"context"
	"io"
)

// FileStorage persists an uploaded file's bytes and returns a URL it can
// later be fetched from. Media.URL is just a string — the domain never
// knows or cares whether that URL points at local disk, S3, or a CDN;
// infrastructure implements this port against whichever backend is
// configured, so swapping storage later touches no domain/application code.
type FileStorage interface {
	Save(ctx context.Context, tenantID, filename string, data io.Reader) (url string, err error)
	Delete(ctx context.Context, url string) error
	// Get re-reads a previously-saved file's bytes back out of storage --
	// needed by the procurement document-parsing job (application/procurement),
	// which runs asynchronously after the upload request that called Save
	// has already finished.
	Get(ctx context.Context, url string) (io.ReadCloser, error)
}
