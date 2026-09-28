package theme

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/gif"  // register decoders for image.DecodeConfig
	_ "image/jpeg" //
	_ "image/png"  //
	"io"
	"net/http"
	"path/filepath"

	"github.com/adirkuhn/commerceos/domain/theme"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type UploadAssetInput struct {
	UserID   string
	Filename string
	Data     io.Reader
	Alt      string
}

// UploadAssetUseCase adds an image to the tenant's theme media library.
// The type is detected from the bytes, never trusted from the filename or
// the client's Content-Type, so e.g. an SVG (which can carry scripts)
// named "logo.png" is refused.
type UploadAssetUseCase struct{ deps Deps }

func NewUploadAssetUseCase(deps Deps) *UploadAssetUseCase { return &UploadAssetUseCase{deps: deps} }

func (uc *UploadAssetUseCase) Execute(ctx context.Context, in UploadAssetInput) (*theme.Asset, error) {
	data, err := io.ReadAll(io.LimitReader(in.Data, theme.MaxAssetBytes+1))
	if err != nil {
		return nil, fmt.Errorf("uploadAsset: read: %w", err)
	}
	contentType := http.DetectContentType(data)
	width, height := 0, 0
	if cfg, _, err := image.DecodeConfig(bytes.NewReader(data)); err == nil {
		width, height = cfg.Width, cfg.Height
	}
	tenantID := tenant.IDFromContext(ctx)
	name := filepath.Base(in.Filename)
	asset, err := theme.NewAsset(tenantID, "", name, contentType, int64(len(data)), width, height, in.Alt, in.UserID, uc.deps.Now())
	if err != nil {
		return nil, fmt.Errorf("uploadAsset: %w", err)
	}
	// Storage is external I/O, kept out of the DB transaction (same
	// trade-off as application/pms.UploadMediaUseCase: a failed insert
	// leaves an orphaned file, never a row without a file).
	if asset.URL, err = uc.deps.Storage.Save(ctx, tenantID, asset.StorageName(), bytes.NewReader(data)); err != nil {
		return nil, fmt.Errorf("uploadAsset: store: %w", err)
	}
	if err := tenant.WithTenant(ctx, uc.deps.DB, func(ctx context.Context) error {
		return uc.deps.Assets.Create(ctx, &asset)
	}); err != nil {
		return nil, fmt.Errorf("uploadAsset: %w", err)
	}
	return &asset, nil
}
