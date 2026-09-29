package theme

import "sort"

// A preset is what a new theme starts from: its documents (every template
// with a working layout, header and footer) and its settings. A test checks
// every preset's documents against the component contract, so a new theme
// is always publishable.
type preset struct {
	documents func() []Document
	settings  func() Settings
}

var presetRegistry = map[string]preset{
	PresetDefault: {documents: defaultDocuments, settings: DefaultSettings},
	PresetMercado: {documents: mercadoDocuments, settings: mercadoSettings},
}

// PresetKeys lists the presets, sorted.
func PresetKeys() []string {
	keys := make([]string, 0, len(presetRegistry))
	for k := range presetRegistry {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func PresetDocuments(presetKey string) ([]Document, error) {
	p, ok := presetRegistry[presetKey]
	if !ok {
		return nil, ErrUnknownPreset
	}
	return p.documents(), nil
}

func PresetSettings(presetKey string) (Settings, error) {
	p, ok := presetRegistry[presetKey]
	if !ok {
		return Settings{}, ErrUnknownPreset
	}
	return p.settings(), nil
}

// templatesFrom builds every template document, taking each one's layout
// from template(key).
func templatesFrom(template func(key string) Node) []Document {
	var docs []Document
	for _, key := range templateKeys {
		docs = append(docs, Document{Kind: KindTemplate, Key: key, Root: template(key)})
	}
	return docs
}

// defaultDocuments: a general-purpose store.
func defaultDocuments() []Document {
	return append(templatesFrom(defaultTemplate),
		Document{Kind: KindSectionGroup, Key: "header", Root: defaultHeader()},
		Document{Kind: KindSectionGroup, Key: "footer", Root: defaultFooter()},
	)
}

func n(id, typ string, props map[string]any, children ...Node) Node {
	return Node{ID: id, Type: typ, Props: props, Children: children}
}

func root(children ...Node) Node {
	r := EmptyRoot()
	r.Children = children
	return r
}

func heading(id, text string, level int) Node {
	return n(id, "heading", map[string]any{"text": text, "level": level})
}

func paragraph(id, text string) Node {
	return n(id, "text", map[string]any{"content": []any{map[string]any{"type": "paragraph", "children": []any{map[string]any{"text": text}}}}})
}

func defaultTemplate(key string) Node {
	switch key {
	case "home":
		return root(
			n("hero-section", "section", map[string]any{"full_width": true},
				n("hero", "hero", map[string]any{"title": "Welcome to our store", "subtitle": "Discover our latest products.",
					"button_label": "Shop now", "button_link": "/search?q=", "align": "center", "height": "lg"})),
			n("collections-section", "section", nil, heading("collections-title", "Shop by collection", 2),
				n("collections", "collection-list", map[string]any{"source": "all", "limit": 6})),
			n("featured", "section", nil, heading("featured-title", "Featured products", 2),
				n("featured-grid", "product-grid", map[string]any{"source": "all", "limit": 8, "show_price": true})),
			n("banner-section", "section", map[string]any{"full_width": true},
				n("promo", "banner", map[string]any{"text": "Free shipping on orders over €50", "link_label": "Learn more", "link": "/pages/shipping"})),
			n("newsletter-section", "section", nil,
				n("newsletter", "newsletter-signup", map[string]any{"title": "Stay in the loop", "text": "News and offers, straight to your inbox.",
					"button_label": "Subscribe", "consent_label": "I'd like to receive emails from this store. I can unsubscribe at any time."})),
		)
	case "product":
		return root(
			n("product-main", "section", nil,
				n("product-crumbs", "breadcrumbs", nil),
				n("product-cols", "columns", map[string]any{"stack_on": "mobile"},
					n("product-media", "column", nil, n("product-gallery", "product-gallery", map[string]any{"layout": "thumbs_below"})),
					n("product-info", "column", nil,
						n("product-title", "product-title", map[string]any{"level": 1}),
						n("product-price", "product-price", map[string]any{"show_compare_at": true}),
						n("product-variants", "variant-selector", map[string]any{"style": "buttons"}),
						n("product-qty", "quantity-selector", nil),
						n("product-add", "add-to-cart", map[string]any{"label": "Add to cart", "sold_out_label": "Sold out"}),
						n("product-wish", "wishlist-button", map[string]any{"add_label": "Save to wishlist", "remove_label": "Saved"}),
						n("product-desc", "product-description", nil),
					),
				),
			),
			n("product-reviews-section", "section", nil, n("product-reviews", "product-reviews", map[string]any{"title": "Reviews", "limit": 10})),
			n("product-recs-section", "section", nil, n("product-recs", "product-recommendations", map[string]any{"title": "You may also like", "limit": 4})),
		)
	case "collection":
		return root(n("collection-main", "section", nil,
			n("collection-crumbs", "breadcrumbs", nil),
			n("collection-header", "collection-header", map[string]any{"show_image": true, "show_description": true}),
			n("collection-filters", "collection-filters", nil),
			n("collection-products", "collection-products", map[string]any{"per_page": 24, "show_price": true}),
			n("collection-pages", "pagination", nil),
		))
	case "search":
		return root(n("search-main", "section", nil,
			n("search-header", "collection-header", map[string]any{"show_image": false, "show_description": false}),
			n("search-products", "collection-products", map[string]any{"per_page": 24, "show_price": true}),
			n("search-pages", "pagination", nil),
		))
	case "cart":
		return root(n("cart-main", "section", nil,
			heading("cart-title", "Your cart", 1),
			n("cart-cols", "columns", map[string]any{"stack_on": "tablet"},
				n("cart-lines", "column", map[string]any{"span": 2}, n("cart-items", "cart-items", map[string]any{"empty_message": "Your cart is empty."})),
				n("cart-side", "column", nil,
					n("cart-summary", "cart-summary", map[string]any{"show_coupon": true}),
					n("cart-checkout", "checkout-button", map[string]any{"label": "Checkout"}),
				),
			),
		))
	case "checkout":
		return root(n("checkout-main", "section", nil, heading("checkout-title", "Checkout", 1),
			n("checkout-form", "checkout-form", map[string]any{"button_label": "Place order"})))
	case "order-confirmation":
		return root(n("order-main", "section", nil, n("order-status", "order-status", map[string]any{"title": "Thank you for your order"})))
	case "login":
		return root(n("login-main", "section", nil, n("login-form", "login-form", map[string]any{"title": "Sign in", "button_label": "Email me a sign-in link"})))
	case "account":
		return root(n("account-main", "section", nil, heading("account-title", "Your account", 1),
			n("account-overview", "account-overview", map[string]any{"show_orders": true})))
	case "wishlist":
		return root(n("wishlist-main", "section", nil, heading("wishlist-title", "Your wishlist", 1),
			n("wishlist-items", "wishlist-items", map[string]any{"empty_message": "Nothing saved yet."})))
	case "not-found":
		return root(n("nf", "section", nil,
			heading("nf-title", "Page not found", 1),
			paragraph("nf-text", "The page you're looking for doesn't exist."),
			n("nf-home", "button", map[string]any{"label": "Back to the shop", "link": "/"}),
		))
	}
	return EmptyRoot()
}

func defaultHeader() Node {
	return root(n("header-bar", "section", nil,
		n("header-cols", "columns", map[string]any{"stack_on": "never"},
			n("header-brand", "column", nil, n("logo", "site-logo", map[string]any{"size": "md"})),
			n("header-nav", "column", map[string]any{"span": 2}, n("menu", "navigation-menu", map[string]any{
				"orientation": "horizontal",
				"items":       []any{map[string]any{"label": "Home", "link": "/"}, map[string]any{"label": "Search", "link": "/search?q="}},
			})),
			n("header-actions", "column", nil,
				n("search", "search-box", map[string]any{"placeholder": "Search products"}),
				n("account", "account-link", map[string]any{"login_label": "Sign in", "account_label": "Account"}),
				n("cart", "cart-link", map[string]any{"label": "Cart", "show_count": true}),
			),
		),
	))
}

func defaultFooter() Node {
	return root(n("footer-bar", "section", nil,
		n("footer-cols", "columns", map[string]any{"stack_on": "mobile"},
			n("footer-brand", "column", nil, n("footer-logo", "site-logo", map[string]any{"size": "sm"})),
			n("footer-links", "column", nil, n("footer-menu", "navigation-menu", map[string]any{
				"orientation": "vertical",
				"items":       []any{map[string]any{"label": "Shipping", "link": "/pages/shipping"}, map[string]any{"label": "Returns", "link": "/pages/returns"}},
			})),
		),
	))
}
