package theme

import (
	"context"
	"errors"
	"io"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

// MaxAssetBytes caps one uploaded theme image.
const MaxAssetBytes = 10 << 20

const maxAltLength = 250

var (
	ErrAssetNotFound        = errors.New("asset not found")
	ErrUnsupportedAssetType = errors.New("only JPEG, PNG, WebP, GIF and ICO images can be uploaded")
	ErrAssetTooLarge        = errors.New("images can be at most 10 MB")
	ErrInvalidAssetAlt      = errors.New("image description must be at most 250 characters")
)

// SVG is deliberately excluded: it can carry scripts. Values are the file
// extension a stored asset gets, so storage serves the right Content-Type.
var assetTypes = map[string]string{
	"image/jpeg": ".jpg", "image/png": ".png", "image/webp": ".webp", "image/gif": ".gif", "image/x-icon": ".ico",
}

// Asset is an image in the tenant's theme media library. Documents and
// settings refer to it as "asset:<id>" and never by URL, so storage can
// move (e.g. behind a CDN) without touching published versions. Deleting
// an asset only hides it from the library: published versions may still
// show it.
type Asset struct {
	ID          string
	TenantID    string
	URL         string
	Filename    string
	ContentType string
	Bytes       int64
	Width       int
	Height      int
	Alt         string
	CreatedBy   string
	CreatedAt   time.Time
	DeletedAt   *time.Time
}

func NewAsset(tenantID, url, filename, contentType string, bytes int64, width, height int, alt, createdBy string, now time.Time) (Asset, error) {
	if _, ok := assetTypes[contentType]; !ok {
		return Asset{}, ErrUnsupportedAssetType
	}
	if bytes > MaxAssetBytes {
		return Asset{}, ErrAssetTooLarge
	}
	a := Asset{
		TenantID: tenantID, URL: url, Filename: filename, ContentType: contentType, Bytes: bytes,
		Width: width, Height: height, CreatedBy: createdBy, CreatedAt: now,
	}
	return a, a.SetAlt(alt)
}

// StorageName is the name the asset's bytes are stored under: the
// extension follows the detected type, never the uploaded filename.
func (a Asset) StorageName() string { return "asset" + assetTypes[a.ContentType] }

func (a *Asset) SetAlt(alt string) error {
	alt = strings.TrimSpace(alt)
	if utf8.RuneCountInString(alt) > maxAltLength {
		return ErrInvalidAssetAlt
	}
	a.Alt = alt
	return nil
}

// AssetStorage stores the bytes of an uploaded asset and returns the URL
// it's publicly served from.
type AssetStorage interface {
	Save(ctx context.Context, tenantID, filename string, data io.Reader) (url string, err error)
}

type AssetRepository interface {
	// Create fills in a.ID.
	Create(ctx context.Context, a *Asset) error
	// FindByID returns ErrAssetNotFound when missing (deleted ones included).
	FindByID(ctx context.Context, id string) (*Asset, error)
	// ListActive lists the library (not deleted), newest first.
	ListActive(ctx context.Context, limit, offset int) ([]Asset, int64, error)
	// FindByIDs resolves references -- deleted assets included, since a
	// published version may still show them.
	FindByIDs(ctx context.Context, ids []string) ([]Asset, error)
	// Update persists Alt and DeletedAt.
	Update(ctx context.Context, a *Asset) error
}

var assetRefPattern = regexp.MustCompile(`^asset:([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})$`)

// ReferencedAssetIDs lists every asset a node tree refers to, in props
// (any string value, however nested) and styles.
func ReferencedAssetIDs(root Node) []string {
	seen := map[string]bool{}
	var out []string
	add := func(s string) {
		if m := assetRefPattern.FindStringSubmatch(s); m != nil && !seen[m[1]] {
			seen[m[1]] = true
			out = append(out, m[1])
		}
	}
	var walkValue func(v any)
	walkValue = func(v any) {
		switch x := v.(type) {
		case string:
			add(x)
		case []any:
			for _, e := range x {
				walkValue(e)
			}
		case map[string]any:
			for _, e := range x {
				walkValue(e)
			}
		}
	}
	var walk func(n Node)
	walk = func(n Node) {
		walkValue(map[string]any(n.Props))
		for _, bp := range []map[string]string{n.Style.Base, n.Style.Tablet, n.Style.Mobile} {
			for _, v := range bp {
				add(v)
			}
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(root)
	return out
}

// ReferencedAssetIDs lists the assets the settings refer to (logo, favicon).
func (s Settings) ReferencedAssetIDs() []string {
	var out []string
	for _, id := range []*string{s.Brand.LogoAssetID, s.Brand.FaviconAssetID} {
		if id != nil && *id != "" {
			out = append(out, *id)
		}
	}
	return out
}
