package messagecenter

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"

	mcDomain "github.com/adirkuhn/commerceos/domain/messagecenter"
	"github.com/adirkuhn/commerceos/internal/idgen"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// imageStorage is pms.FileStorage narrowed to point of use.
type imageStorage interface {
	Save(ctx context.Context, tenantID, filename string, data io.Reader) (url string, err error)
}

const maxTemplateImageBytes = 5 << 20

var templateImageExtensions = map[string]string{
	"image/jpeg": "jpg", "image/png": "png", "image/gif": "gif", "image/webp": "webp",
}

// UploadTemplateImageUseCase stores an image a message template shows (a
// logo, a banner) and returns its public URL. The type is detected from
// the bytes, never trusted from the filename.
type UploadTemplateImageUseCase struct {
	storage imageStorage
}

func NewUploadTemplateImageUseCase(storage imageStorage) *UploadTemplateImageUseCase {
	return &UploadTemplateImageUseCase{storage: storage}
}

func (uc *UploadTemplateImageUseCase) Execute(ctx context.Context, file io.Reader) (string, error) {
	data, err := io.ReadAll(io.LimitReader(file, maxTemplateImageBytes+1))
	if err != nil {
		return "", fmt.Errorf("uploadTemplateImage: %w", err)
	}
	if len(data) > maxTemplateImageBytes {
		return "", fmt.Errorf("uploadTemplateImage: %w", mcDomain.ErrTemplateImageTooLarge)
	}
	ext, ok := templateImageExtensions[http.DetectContentType(data)]
	if !ok {
		return "", fmt.Errorf("uploadTemplateImage: %w", mcDomain.ErrTemplateImageTypeUnsupported)
	}
	name := "message-images/" + idgen.NewV4() + "." + ext
	url, err := uc.storage.Save(ctx, tenant.IDFromContext(ctx), name, bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("uploadTemplateImage: %w", err)
	}
	return url, nil
}
