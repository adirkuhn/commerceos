package theme

// PresetDocuments returns the documents a new theme's first draft starts
// with. The default preset gives every template a working layout -- the
// commerce components each page needs -- and empty header/footer/other
// pages to design from. A test checks every preset document against the
// component contract, so a new theme is always publishable.
func PresetDocuments(presetKey string) ([]Document, error) {
	if !presets[presetKey] {
		return nil, ErrUnknownPreset
	}
	var docs []Document
	for _, key := range templateKeys {
		docs = append(docs, Document{Kind: KindTemplate, Key: key, Root: defaultTemplate(key)})
	}
	for _, key := range sectionGroupKeys {
		docs = append(docs, Document{Kind: KindSectionGroup, Key: key, Root: EmptyRoot()})
	}
	return docs, nil
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
			n("hero", "section", nil, heading("hero-title", "Welcome to our store", 1), paragraph("hero-text", "Discover our latest products.")),
			n("featured", "section", nil, heading("featured-title", "Featured products", 2),
				n("featured-grid", "product-grid", map[string]any{"source": "all", "limit": 8, "show_price": true})),
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
	case "not-found":
		return root(n("nf", "section", nil,
			heading("nf-title", "Page not found", 1),
			paragraph("nf-text", "The page you're looking for doesn't exist."),
			n("nf-home", "button", map[string]any{"label": "Back to the shop", "link": "/"}),
		))
	}
	return EmptyRoot()
}
