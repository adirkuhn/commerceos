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

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
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
	asset, err := uc.deps.storeAsset(ctx, in.Filename, data, in.Alt, in.UserID)
	if err != nil {
		return nil, fmt.Errorf("uploadAsset: %w", err)
	}
	return asset, nil
}

// storeAsset adds image bytes to the tenant's media library: detects the
// type, saves the file, then records the asset.
func (d Deps) storeAsset(ctx context.Context, filename string, data []byte, alt, userID string) (*theme.Asset, error) {
	contentType := http.DetectContentType(data)
	width, height := 0, 0
	if cfg, _, err := image.DecodeConfig(bytes.NewReader(data)); err == nil {
		width, height = cfg.Width, cfg.Height
	}
	tenantID := tenant.IDFromContext(ctx)
	name := filepath.Base(filename)
	asset, err := theme.NewAsset(tenantID, "", name, contentType, int64(len(data)), width, height, alt, userID, d.Now())
	if err != nil {
		return nil, fmt.Errorf("storeAsset: %w", err)
	}
	// Storage is external I/O, kept out of the DB transaction (same
	// trade-off as application/pms.UploadMediaUseCase: a failed insert
	// leaves an orphaned file, never a row without a file).
	if asset.URL, err = d.Storage.Save(ctx, tenantID, asset.StorageName(), bytes.NewReader(data)); err != nil {
		return nil, fmt.Errorf("storeAsset: save: %w", err)
	}
	if err := tenant.WithTenant(ctx, d.DB, func(ctx context.Context) error {
		if err := d.Assets.Create(ctx, &asset); err != nil {
			return err
		}
		return eventrecord.Record(ctx, d.Outbox, events.ThemeAssetUploaded, asset.ID, eventrecord.Data{
			"filename": asset.Filename, "content_type": asset.ContentType, "bytes": asset.Bytes,
			"width": asset.Width, "height": asset.Height,
		})
	}); err != nil {
		return nil, fmt.Errorf("storeAsset: %w", err)
	}
	return &asset, nil
}
