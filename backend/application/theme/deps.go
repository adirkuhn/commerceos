// Package theme holds the storefront theme use cases: the theme library,
// draft editing, publish/rollback, editor/preview sessions and the public
// (shopper-facing) reads. See docs/storefront-architecture.md.
package theme

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/domain/theme"
	"github.com/adirkuhn/commerceos/internal/idgen"
)

type channelFinder interface {
	FindByID(ctx context.Context, id string) (*channelsDomain.SalesChannel, error)
}

type outboxEnqueuer interface {
	Enqueue(ctx context.Context, subject, eventID string, payload []byte, occurredAt time.Time) error
}

// Deps is everything the theme use cases are built from. Each use case
// takes only what it needs from it; one struct keeps the wiring in one
// place instead of fifteen long constructor argument lists.
type Deps struct {
	DB       *gorm.DB
	Channels channelFinder
	Themes   theme.ThemeRepository
	Versions theme.VersionRepository
	Docs     theme.DocumentRepository
	Sessions theme.SessionRepository
	Outbox   outboxEnqueuer
	Catalog  *theme.Catalog
	Assets   theme.AssetRepository
	Storage  theme.AssetStorage
	// PresetImages serves the pictures presets ship with; nil leaves new
	// themes without them.
	PresetImages presetImageReader
	// NewToken returns a fresh, unguessable bearer token.
	NewToken func() (string, error)
	Now      func() time.Time
}

func (d Deps) requireThemedChannel(ctx context.Context, channelID string) (*channelsDomain.SalesChannel, error) {
	ch, err := d.Channels.FindByID(ctx, channelID)
	if err != nil {
		return nil, err
	}
	if !ch.HostsThemedStorefront() {
		return nil, theme.ErrStorefrontNotEnabled
	}
	return ch, nil
}

func (d Deps) enqueue(ctx context.Context, event events.DomainEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal %s: %w", event.EventName(), err)
	}
	return d.Outbox.Enqueue(ctx, event.EventName(), event.GetEventID(), payload, event.GetOccurredAt())
}

func (d Deps) eventBase() events.Base {
	return events.Base{EventID: idgen.NewV4(), OccurredAt: d.Now()}
}

// lockedDraft loads the theme's draft for writing, failing with ErrNoDraft
// when the editor hasn't opened one.
func (d Deps) lockedDraft(ctx context.Context, themeID string) (*theme.Version, error) {
	if _, err := d.Themes.FindByID(ctx, themeID); err != nil {
		return nil, err
	}
	draft, err := d.Versions.FindDraft(ctx, themeID, true)
	if err != nil {
		return nil, err
	}
	if draft == nil {
		return nil, theme.ErrNoDraft
	}
	return draft, nil
}

// pageWindow turns page/per_page into limit/offset, clamping per_page to
// 1..100 and page to >= 1.
func pageWindow(page, perPage int) (limit, offset int) {
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}
	if page < 1 {
		page = 1
	}
	return perPage, (page - 1) * perPage
}

// previewVersion is the version a preview shows: the draft when there is
// one, otherwise the theme's published version.
func (d Deps) previewVersion(ctx context.Context, salesChannelID, themeID string) (*theme.Theme, *theme.Version, error) {
	th, err := d.Themes.FindByID(ctx, themeID)
	if err != nil {
		return nil, nil, err
	}
	// A preview token is only honoured on its own theme's storefront.
	if th.SalesChannelID != salesChannelID {
		return nil, nil, theme.ErrThemeNotFound
	}
	v, err := d.Versions.FindDraft(ctx, themeID, false)
	if err != nil {
		return nil, nil, err
	}
	if v == nil {
		if th.PublishedVersionID == nil {
			return nil, nil, theme.ErrVersionNotFound
		}
		if v, err = d.Versions.FindByID(ctx, *th.PublishedVersionID); err != nil {
			return nil, nil, err
		}
	}
	return th, v, nil
}

// resolveAssets looks up the referenced assets (deleted ones included --
// a published version may still show them), keyed by id. Unknown ids are
// left out; the storefront renders nothing for them.
func (d Deps) resolveAssets(ctx context.Context, ids []string) (map[string]theme.Asset, error) {
	out := map[string]theme.Asset{}
	if len(ids) == 0 {
		return out, nil
	}
	found, err := d.Assets.FindByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	for _, a := range found {
		out[a.ID] = a
	}
	return out, nil
}

// DocumentView is a document plus the assets it refers to.
type DocumentView struct {
	theme.Document
	Assets map[string]theme.Asset
}

func (d Deps) documentView(ctx context.Context, doc *theme.Document) (*DocumentView, error) {
	assets, err := d.resolveAssets(ctx, theme.ReferencedAssetIDs(doc.Root))
	if err != nil {
		return nil, err
	}
	return &DocumentView{Document: *doc, Assets: assets}, nil
}
