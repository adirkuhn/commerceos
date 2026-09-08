// Package theme holds the visual page-builder document model for one
// internal storefront channel -- a ThemePage is a free-form canvas of
// absolutely-positioned Blocks (hero banners, product grids, rich text,
// custom HTML/CSS), edited via a drag/resize canvas in the admin app and
// rendered as-is by the storefront-frontend app. Deliberately a single
// JSONB document per page rather than normalized block/section tables --
// same "whole document, rewritten wholesale" convention as
// domain/storefront.Cart and domain/whatsapp.Conversation's own JSONB
// fields: a page's block list has no independent existence outside its
// page, and is always edited/saved as one unit from the canvas editor.
package theme

import "time"

// BlockType is the fixed set of block kinds the canvas editor can place --
// each has its own Props shape (validated at the application layer, not
// here) and its own renderer in the storefront-frontend app.
type BlockType string

const (
	BlockTypeHero        BlockType = "hero"
	BlockTypeProductGrid BlockType = "product_grid"
	BlockTypeRichText    BlockType = "rich_text"
	BlockTypeCustomHTML  BlockType = "custom_html"
)

func (t BlockType) Valid() bool {
	switch t {
	case BlockTypeHero, BlockTypeProductGrid, BlockTypeRichText, BlockTypeCustomHTML:
		return true
	default:
		return false
	}
}

// Rect is one block's absolute position/size on the canvas, in pixels.
// Canvas coordinates are relative to ThemePage.CanvasWidth -- the
// storefront-frontend scales the whole canvas to fit the visitor's
// viewport rather than each block scaling independently, same as how
// Figma/Webflow-style canvases render a fixed-width design responsively.
type Rect struct {
	X      int
	Y      int
	Width  int
	Height int
}

// BlockLayout is a block's position, one Rect per breakpoint -- Desktop is
// the only breakpoint implemented today (single-breakpoint v1, see the
// package doc comment); Mobile is already a field so a later increment
// adding per-breakpoint layouts is additive (a new column default, not a
// breaking migration). A block with no Mobile override falls back to
// Desktop when that increment lands.
type BlockLayout struct {
	Desktop Rect
	Mobile  *Rect
}

// Block is one placed element on a ThemePage's canvas. Props is
// block-type-specific free-form data (e.g. Hero's heading text/image URL,
// ProductGrid's category filter, CustomHTML's raw markup/CSS) --
// deliberately untyped here since each BlockType owns its own Props
// schema at the application/rendering layer, not the domain layer.
type Block struct {
	ID     string
	Type   BlockType
	Layout BlockLayout
	Props  map[string]any
}

// ThemePage is one page's whole canvas document for one sales channel --
// "home" is the only Slug used today (the storefront's landing page); the
// field exists so a later increment adding more pages (product listing,
// about, etc.) is additive. CanvasWidth is the fixed design width every
// Block's Rect is positioned against (see Rect's doc comment).
type ThemePage struct {
	ID             string
	TenantID       string
	SalesChannelID string
	Slug           string
	CanvasWidth    int
	Blocks         []Block
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
