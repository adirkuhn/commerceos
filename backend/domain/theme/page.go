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
//
// Functional block types (ProductDetails/CartSummary/CheckoutForm/
// OrderStatus, see IsFunctional) render real wired-up behavior -- the
// actual add-to-cart control, the actual payment-submitting checkout form
// -- not just presentation, unlike Hero/RichText/ProductGrid/CustomHTML.
// A merchant can drag/resize/restyle one of these like any other block,
// but PageSlug's own doc comment on RequiredBlockType explains what
// happens if one goes missing from a page that needs it: the
// storefront-frontend renders a safe default in its place rather than
// ever shipping a broken cart/checkout/product page.
type BlockType string

const (
	BlockTypeHero             BlockType = "hero"
	BlockTypeProductGrid      BlockType = "product_grid"
	BlockTypeFeaturedProducts BlockType = "featured_products"
	BlockTypeRichText         BlockType = "rich_text"
	BlockTypeCustomHTML       BlockType = "custom_html"
	// BlockTypeNavMenu is a merchant-ordered list of label/url links -- not
	// tied to any one concept (categories, legal pages, ...), so it's
	// reusable on both the header page (category browse links) and the
	// footer page (privacy/terms/returns links) without needing two
	// separate block types.
	BlockTypeNavMenu BlockType = "nav_menu"

	BlockTypeProductDetails BlockType = "product_details"
	BlockTypeCartSummary    BlockType = "cart_summary"
	BlockTypeCheckoutForm   BlockType = "checkout_form"
	BlockTypeOrderStatus    BlockType = "order_status"
	BlockTypeWishlist       BlockType = "wishlist"
	BlockTypeLoginForm      BlockType = "login_form"
	BlockTypeAccount        BlockType = "account"
	// BlockTypeNavActions is the header page's one required functional
	// block -- the real, live Cart/Wishlist/Account/Track-order links
	// (real counts, real auth state), bundled as one wired widget the same
	// way BlockTypeCheckoutForm bundles a whole form rather than being
	// split into one block per field. See PageSlugHeader's own
	// RequiredBlockType entry.
	BlockTypeNavActions BlockType = "nav_actions"
)

func (t BlockType) Valid() bool {
	switch t {
	case BlockTypeHero, BlockTypeProductGrid, BlockTypeFeaturedProducts, BlockTypeRichText, BlockTypeCustomHTML, BlockTypeNavMenu,
		BlockTypeProductDetails, BlockTypeCartSummary, BlockTypeCheckoutForm, BlockTypeOrderStatus, BlockTypeWishlist,
		BlockTypeLoginForm, BlockTypeAccount, BlockTypeNavActions:
		return true
	default:
		return false
	}
}

// IsFunctional reports whether this block type renders real wired-up
// behavior rather than pure presentation -- see the type's own doc
// comment.
func (t BlockType) IsFunctional() bool {
	switch t {
	case BlockTypeProductDetails, BlockTypeCartSummary, BlockTypeCheckoutForm, BlockTypeOrderStatus, BlockTypeWishlist,
		BlockTypeLoginForm, BlockTypeAccount, BlockTypeNavActions:
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

// PageSlug is the fixed set of storefront pages that can have their own
// canvas document -- every one of them is canvas-editable (including Cart
// and Checkout), each via its own RequiredBlockType so its real behavior
// survives however the merchant lays it out.
type PageSlug string

const (
	PageSlugHome              PageSlug = "home"
	PageSlugProducts          PageSlug = "products"
	PageSlugProduct           PageSlug = "product"
	PageSlugCart              PageSlug = "cart"
	PageSlugCheckout          PageSlug = "checkout"
	PageSlugOrderConfirmed    PageSlug = "order-confirmation"
	PageSlugOrderStatusLookup PageSlug = "order-lookup"
	PageSlugWishlist          PageSlug = "wishlist"
	PageSlugLogin             PageSlug = "login"
	PageSlugAccount           PageSlug = "account"
	// PageSlugHeader/PageSlugFooter are the two site-wide "pseudo-pages" --
	// not a real storefront route (a shopper never navigates to
	// /store/header), just the same ThemePage/canvas-editor/embedded-Puck
	// machinery every real page already gets, reused so header/footer
	// content is a customizable block layout too instead of hardcoded
	// chrome. storefront-frontend's Layout renders them around every real
	// page's own content.
	PageSlugHeader PageSlug = "header"
	PageSlugFooter PageSlug = "footer"
)

func (s PageSlug) Valid() bool {
	switch s {
	case PageSlugHome, PageSlugProducts, PageSlugProduct, PageSlugCart, PageSlugCheckout,
		PageSlugOrderConfirmed, PageSlugOrderStatusLookup, PageSlugWishlist, PageSlugLogin, PageSlugAccount,
		PageSlugHeader, PageSlugFooter:
		return true
	default:
		return false
	}
}

// RequiredBlockType is the one functional block a page needs to actually
// work -- ok is false for Home/Products, which are pure presentation and
// need no functional block at all. The storefront-frontend renders this
// block type's safe default UI when a required page is missing it (a
// merchant deleted it, or the page was never customized), so a page can
// always be laid out freely without ever being capable of shipping
// without its real behavior.
func (s PageSlug) RequiredBlockType() (BlockType, bool) {
	switch s {
	case PageSlugProduct:
		return BlockTypeProductDetails, true
	case PageSlugCart:
		return BlockTypeCartSummary, true
	case PageSlugCheckout:
		return BlockTypeCheckoutForm, true
	case PageSlugOrderConfirmed, PageSlugOrderStatusLookup:
		return BlockTypeOrderStatus, true
	case PageSlugWishlist:
		return BlockTypeWishlist, true
	case PageSlugLogin:
		return BlockTypeLoginForm, true
	case PageSlugAccount:
		return BlockTypeAccount, true
	case PageSlugHeader:
		return BlockTypeNavActions, true
	default:
		return "", false
	}
}

// EditorMode selects which document a ThemePage actually renders from --
// Blocks (ModeFreeform, the absolutely-positioned canvas the rest of this
// package's doc comments describe) or PuckData (ModePuck, an opaque
// document authored by storefront-frontend's own embedded Puck editor,
// which renders the real live storefront components directly -- see
// PuckData's own doc comment for why). Both documents are kept on every
// row regardless of which is active: switching modes back and forth never
// loses the other one's content.
type EditorMode string

const (
	EditorModeFreeform EditorMode = "freeform"
	EditorModePuck     EditorMode = "puck"
)

func (m EditorMode) Valid() bool {
	switch m {
	case EditorModeFreeform, EditorModePuck:
		return true
	default:
		return false
	}
}

// ThemePage is one page's whole canvas document for one sales channel --
// Slug identifies which storefront page this is (see PageSlug).
// CanvasWidth is the fixed design width every Block's Rect is positioned
// against (see Rect's doc comment). CustomCSS is raw, merchant-authored
// CSS injected into the storefront-frontend's page as-is (same trust
// boundary as CustomHTML's raw markup -- admin-JWT-gated, never shopper
// input) -- an escape hatch for styling the canvas can't express, not a
// replacement for it.
//
// PuckData is EditorMode Puck's document -- untyped here on purpose, same
// reason Block.Props is untyped: its shape belongs to storefront-frontend's
// own Puck component-field configuration, not this domain layer. Only
// meaningful when EditorMode is Puck; nil otherwise.
type ThemePage struct {
	ID             string
	TenantID       string
	SalesChannelID string
	Slug           PageSlug
	CanvasWidth    int
	Blocks         []Block
	CustomCSS      string
	EditorMode     EditorMode
	PuckData       map[string]any
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// HasRequiredBlock reports whether p actually contains the functional
// block its own Slug requires -- true for a page with no requirement at
// all (Home/Products). Used by application/theme's page-fetch use case to
// decide whether a caller should be told a default fallback is in effect.
func (p ThemePage) HasRequiredBlock() bool {
	required, ok := p.Slug.RequiredBlockType()
	if !ok {
		return true
	}
	for _, b := range p.Blocks {
		if b.Type == required {
			return true
		}
	}
	return false
}
