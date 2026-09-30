package theme

// The "fresh" preset: an online supermarket -- green brand color, hero,
// promo cards, category tiles, rows of products and service promises.
// Static layout: no sliders, nothing animating in or out.

func freshSettings() Settings {
	s := DefaultSettings()
	s.Colors = ColorSettings{
		Primary: "#0aad0a", Secondary: "#001e2b", Background: "#ffffff", Surface: "#f0f3f2",
		Text: "#001e2b", Muted: "#5c6c75", Accent: "#eabc49",
	}
	s.Buttons = ButtonSettings{Radius: "md", Size: "md", Style: "solid"}
	s.Product = ProductSettings{GridColumns: 5, ImageRatio: "1:1"}
	s.Header.Sticky = true
	return s
}

func freshDocuments() []Document {
	return append(templatesFrom(freshTemplate),
		Document{Kind: KindSectionGroup, Key: "header", Root: freshHeader()},
		Document{Kind: KindSectionGroup, Key: "footer", Root: freshFooter()},
	)
}

// card is a rounded panel on the surface color.
func card(node Node) Node {
	node.Style = NodeStyle{Base: map[string]string{"background_color": "surface", "border_radius": "lg", "padding_x": "lg", "padding_y": "lg"}}
	return node
}

// promo is a card with a title, a line of text and a "Shop now" button.
func promo(id, title, text string) Node {
	return card(n(id, "column", nil,
		heading(id+"-title", title, 3),
		paragraph(id+"-text", text),
		n(id+"-button", "button", map[string]any{"label": "Shop now", "link": "/search?q=", "variant": "primary"}),
	))
}

// promise is one of the service promises under the products.
func promise(id, title, text string) Node {
	return n(id, "column", nil, heading(id+"-title", title, 3), paragraph(id+"-text", text))
}

func freshTemplate(key string) Node {
	switch key {
	case "home":
		return root(
			n("hero-section", "section", nil,
				card(n("hero", "hero", map[string]any{"title": "Fresh groceries, delivered today",
					"subtitle":     "Opening sale: up to 50% off fruit, vegetables and more.",
					"button_label": "Shop now", "button_link": "/search?q=", "align": "left", "height": "md"}))),
			n("promos-section", "section", nil,
				n("promos", "columns", map[string]any{"stack_on": "tablet"},
					promo("promo-produce", "Fruits & vegetables", "Up to 30% off this week."),
					promo("promo-bakery", "Freshly baked", "Bread and pastries every morning."),
					promo("promo-pantry", "Pantry staples", "Stock up and save on the basics."),
				)),
			n("categories-section", "section", nil,
				heading("categories-title", "Shop popular categories", 2),
				n("categories", "collection-list", map[string]any{"source": "all", "limit": 12}),
			),
			n("banners-section", "section", nil,
				n("banners", "columns", map[string]any{"stack_on": "tablet"},
					promo("banner-fruit", "Fruits & vegetables", "Get up to 30% off."),
					promo("banner-buns", "Freshly baked buns", "Get up to 25% off."),
				)),
			productRow("best", "Daily best sells", "/search?q="),
			productRow("popular", "Popular products", "/search?q="),
			n("promises-section", "section", nil,
				n("promises", "columns", map[string]any{"stack_on": "tablet"},
					promise("promise-delivery", "Fast delivery", "Your order at your door within hours."),
					promise("promise-range", "Wide assortment", "Everything for your kitchen and home in one place."),
					promise("promise-prices", "Best prices & offers", "Fair prices, and new deals every week."),
					promise("promise-returns", "Easy returns", "Not happy with a product? We refund it."),
				)),
			n("newsletter-section", "section", nil,
				n("newsletter", "newsletter-signup", map[string]any{"title": "Get the weekly deals", "text": "New products and offers, straight to your inbox.",
					"button_label": "", "consent_label": ""})),
		)
	case "collection", "search":
		return mercadoTemplate(key)
	}
	return defaultTemplate(key)
}

func freshHeader() Node {
	return root(
		n("announcement-section", "section", map[string]any{"full_width": true},
			n("announcement", "banner", map[string]any{"text": "Super value deals on everything · Free delivery on orders over €50"})),
		n("header-main", "section", nil,
			n("header-cols", "columns", map[string]any{"stack_on": "mobile"},
				n("header-brand", "column", nil, n("logo", "site-logo", map[string]any{"size": "lg"})),
				n("header-search", "column", map[string]any{"span": 2}, n("search", "search-box", map[string]any{"placeholder": ""})),
				n("header-actions", "column", nil,
					n("languages", "language-switcher", map[string]any{"display": "codes"}),
					n("account", "account-link", map[string]any{"login_label": "", "account_label": ""}),
					n("wishlist-link", "navigation-menu", map[string]any{"orientation": "horizontal",
						"items": []any{map[string]any{"label": "Wishlist", "link": "/wishlist"}}}),
					n("cart", "cart-link", map[string]any{"label": "", "show_count": true}),
				),
			),
		),
		n("header-departments", "section", nil,
			n("departments", "navigation-menu", map[string]any{"orientation": "horizontal", "items": []any{
				map[string]any{"label": "Home", "link": "/"},
				map[string]any{"label": "Fruits & vegetables", "link": "/search?q="},
				map[string]any{"label": "Dairy & eggs", "link": "/search?q="},
				map[string]any{"label": "Bakery", "link": "/search?q="},
				map[string]any{"label": "Snacks", "link": "/search?q="},
				map[string]any{"label": "Drinks", "link": "/search?q="},
				map[string]any{"label": "Household", "link": "/search?q="},
			}})),
	)
}

func freshFooter() Node {
	return root(n("footer-main", "section", nil,
		n("footer-cols", "columns", map[string]any{"stack_on": "mobile"},
			n("footer-about", "column", nil,
				n("footer-logo", "site-logo", map[string]any{"size": "md"}),
				paragraph("footer-about-text", "Fresh groceries and everyday essentials, delivered to your door."),
			),
			n("footer-categories", "column", nil,
				heading("footer-categories-title", "Categories", 3),
				n("footer-categories-menu", "navigation-menu", map[string]any{"orientation": "vertical", "items": []any{
					map[string]any{"label": "Fruits & vegetables", "link": "/search?q="},
					map[string]any{"label": "Dairy & eggs", "link": "/search?q="},
					map[string]any{"label": "Bakery", "link": "/search?q="},
					map[string]any{"label": "Drinks", "link": "/search?q="},
				}}),
			),
			n("footer-shoppers", "column", nil,
				heading("footer-shoppers-title", "For shoppers", 3),
				n("footer-shoppers-menu", "navigation-menu", map[string]any{"orientation": "vertical", "items": []any{
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
					map[string]any{"network": "instagram", "link": "https://instagram.com/"},
					map[string]any{"network": "facebook", "link": "https://facebook.com/"},
					map[string]any{"network": "whatsapp", "link": "https://wa.me/"},
				}}),
			),
		),
	))
}
