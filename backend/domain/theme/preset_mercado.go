package theme

// The "mercado" preset: a grocery / market storefront. Brand-neutral --
// the merchant sets logo, colors and texts; product rows show every
// product until pointed at a collection in the editor.

func mercadoSettings() Settings {
	s := DefaultSettings()
	s.Colors = ColorSettings{
		Primary: "#2e7d32", Secondary: "#1b5e20", Background: "#ffffff", Surface: "#f3f6f2",
		Text: "#1c2a1d", Muted: "#667067", Accent: "#f9a825",
	}
	s.Typography.HeadingFont = "system-rounded"
	s.Buttons = ButtonSettings{Radius: "full", Size: "md", Style: "solid"}
	s.Product = ProductSettings{GridColumns: 5, ImageRatio: "1:1"}
	s.Header.Sticky = true
	return s
}

func mercadoDocuments() []Document {
	return append(templatesFrom(mercadoTemplate),
		Document{Kind: KindSectionGroup, Key: "header", Root: mercadoHeader()},
		Document{Kind: KindSectionGroup, Key: "footer", Root: mercadoFooter()},
	)
}

// productRow is a titled row of products with a "See all" link: the
// building block of a market home page.
func productRow(id, title, link string) Node {
	return n(id, "section", nil,
		n(id+"-head", "columns", map[string]any{"stack_on": "never"},
			n(id+"-title-col", "column", map[string]any{"span": 3}, heading(id+"-title", title, 2)),
			n(id+"-link-col", "column", nil, n(id+"-link", "button", map[string]any{"label": "See all", "link": link, "variant": "outline"})),
		),
		n(id+"-grid", "product-grid", map[string]any{
			"source": "all", "limit": 10, "columns": 5,
			"show_price": true, "show_discount": true, "show_stock": true, "show_add_to_cart": true,
		}),
	)
}

func mercadoTemplate(key string) Node {
	switch key {
	case "home":
		return root(
			n("hero-section", "section", map[string]any{"full_width": true},
				n("hero", "hero", map[string]any{"title": "Big sale: up to 60% off", "subtitle": "Your favourite groceries, delivered to your door.",
					"button_label": "Shop the deals", "button_link": "/search?q=", "align": "left", "height": "md"})),
			productRow("deals", "Deals of the week", "/search?q="),
			productRow("snacks", "Snacks", "/search?q="),
			productRow("favourites", "Customer favourites", "/search?q="),
			productRow("drinks", "Drinks", "/search?q="),
			productRow("new", "New arrivals", "/search?q="),
			n("trust-section", "section", map[string]any{"full_width": true},
				n("trust", "banner", map[string]any{"text": "100% secure shopping · Fast delivery · Friendly support"})),
			n("newsletter-section", "section", nil,
				n("newsletter", "newsletter-signup", map[string]any{"title": "Get the weekly deals", "text": "New products and offers, straight to your inbox.",
					"button_label": "", "consent_label": ""})),
		)
	case "collection", "search":
		return root(n("listing-main", "section", nil,
			n("crumbs", "breadcrumbs", map[string]any{"home_label": ""}),
			n("listing-header", "collection-header", map[string]any{"show_image": true, "show_description": true}),
			n("listing-cols", "columns", map[string]any{"stack_on": "tablet"},
				n("listing-filters-col", "column", nil, n("filters", "collection-filters", map[string]any{"title": "Categories"})),
				n("listing-products-col", "column", map[string]any{"span": 4},
					n("products", "collection-products", map[string]any{
						"per_page": 24, "columns": 4, "show_price": true, "show_discount": true, "show_stock": true, "show_add_to_cart": true,
					}),
					n("pages", "pagination", nil),
				),
			),
		))
	}
	return defaultTemplate(key)
}

func mercadoHeader() Node {
	return root(
		n("announcement-section", "section", map[string]any{"full_width": true},
			n("announcement", "banner", map[string]any{"text": "Free delivery on orders over €50 · Minimum order €30"})),
		n("header-main", "section", nil,
			n("header-cols", "columns", map[string]any{"stack_on": "mobile"},
				n("header-brand", "column", nil, n("logo", "site-logo", map[string]any{"size": "lg"})),
				n("header-search", "column", map[string]any{"span": 2}, n("search", "search-box", map[string]any{"placeholder": ""})),
				n("header-actions", "column", nil,
					n("account", "account-link", map[string]any{"login_label": "", "account_label": ""}),
					n("wishlist-link", "navigation-menu", map[string]any{"orientation": "horizontal",
						"items": []any{map[string]any{"label": "Wishlist", "link": "/wishlist"}}}),
					n("cart", "cart-link", map[string]any{"label": "", "show_count": true}),
				),
			),
		),
		n("header-categories", "section", nil,
			n("categories", "navigation-menu", map[string]any{"orientation": "horizontal", "items": []any{
				map[string]any{"label": "Groceries", "link": "/search?q="},
				map[string]any{"label": "Drinks", "link": "/search?q="},
				map[string]any{"label": "Snacks", "link": "/search?q="},
				map[string]any{"label": "Frozen", "link": "/search?q="},
				map[string]any{"label": "Household", "link": "/search?q="},
				map[string]any{"label": "Personal care", "link": "/search?q="},
			}})),
	)
}

func mercadoFooter() Node {
	return root(n("footer-main", "section", nil,
		n("footer-cols", "columns", map[string]any{"stack_on": "mobile"},
			n("footer-about", "column", nil,
				n("footer-logo", "site-logo", map[string]any{"size": "md"}),
				paragraph("footer-about-text", "The best products, with love, straight to your home."),
			),
			n("footer-contact", "column", nil,
				heading("footer-contact-title", "Contact", 3),
				paragraph("footer-contact-text", "Opening hours: 10:00 – 18:00"),
			),
			n("footer-links", "column", nil,
				heading("footer-links-title", "Help", 3),
				n("footer-menu", "navigation-menu", map[string]any{"orientation": "vertical", "items": []any{
					map[string]any{"label": "My account", "link": "/account"},
					map[string]any{"label": "Wishlist", "link": "/wishlist"},
					map[string]any{"label": "Returns", "link": "/pages/returns"},
					map[string]any{"label": "Privacy policy", "link": "/pages/privacy"},
					map[string]any{"label": "Terms & conditions", "link": "/pages/terms"},
				}}),
			),
			n("footer-social", "column", nil,
				heading("footer-social-title", "Follow us", 3),
				n("social", "social-links", map[string]any{"size": "md", "show_labels": false, "items": []any{
					map[string]any{"network": "whatsapp", "link": "https://wa.me/"},
					map[string]any{"network": "instagram", "link": "https://instagram.com/"},
					map[string]any{"network": "facebook", "link": "https://facebook.com/"},
				}}),
			),
		),
	))
}
