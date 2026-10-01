package theme

// The "fresh" preset: an online supermarket laid out like a classic
// grocery template -- a grey top bar with the account/wishlist/cart icons,
// logo + search + department menu, a large hero, round category pictures,
// two offer banners, product cards with "+ Add", a dark promo beside the
// day's best sellers, bordered service promises with icons and a
// four-column grey footer. Static: no sliders, nothing animating. Its
// look comes from the "supermarket" style (storefront/src/theme/styles).

func freshSettings() Settings {
	s := DefaultSettings()
	s.Style = StyleSupermarket
	s.Colors = ColorSettings{
		Primary: "#0aad0a", Secondary: "#001e2b", Background: "#ffffff", Surface: "#f0f3f2",
		Text: "#001e2b", Muted: "#5c6c75", Accent: "#ffc107",
	}
	s.Typography = TypographySettings{Source: FontSourceGoogle, HeadingFont: "Inter", BodyFont: "Inter", Scale: "sm"}
	s.Buttons = ButtonSettings{Radius: "sm", Size: "sm", Style: "solid"}
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

func styled(node Node, base map[string]string) Node {
	node.Style = NodeStyle{Base: base}
	return node
}

// offer is a banner card: title, a line of text and a "Shop now" button,
// on the surface colour (a picture can be set as its background).
func offer(id, title, text, variant string) Node {
	return styled(n(id, "column", nil,
		heading(id+"-title", title, 3),
		paragraph(id+"-text", text),
		n(id+"-button", "button", map[string]any{"label": "Shop now", "link": "/search?q=", "variant": variant}),
	), map[string]string{"background_color": "surface", "border_radius": "md", "padding_x": "lg", "padding_y": "xl", "gap": "sm"})
}

// darkOffer is an offer on the navy secondary colour with white text.
func darkOffer(id, title, text string) Node {
	white := map[string]string{"color": "background"}
	return styled(n(id, "column", nil,
		styled(heading(id+"-title", title, 3), white),
		styled(paragraph(id+"-text", text), white),
		n(id+"-button", "button", map[string]any{"label": "Shop now", "link": "/search?q=", "variant": "primary"}),
	), map[string]string{"background_color": "secondary", "border_radius": "md", "padding_x": "lg", "padding_y": "xl", "gap": "sm"})
}

// promise is one of the bordered service promises: icon, title, text.
func promise(id, icon, title, text string) Node {
	return styled(n(id, "column", nil,
		n(id+"-icon", "icon", map[string]any{"name": icon, "size": "md"}),
		heading(id+"-title", title, 4),
		paragraph(id+"-text", text),
	), map[string]string{"border_width": "thin", "border_color": "surface", "padding_x": "md", "padding_y": "lg", "text_align": "center", "align_items": "center"})
}

func freshGrid(id string, limit, columns int, addButton string) Node {
	return n(id, "product-grid", map[string]any{
		"source": "all", "limit": limit, "columns": columns,
		"show_price": true, "show_discount": true, "show_stock": false, "show_add_to_cart": true,
		"show_category": true, "add_button": addButton,
	})
}

func freshTemplate(key string) Node {
	switch key {
	case "home":
		return root(
			n("hero-section", "section", nil,
				n("hero", "hero", map[string]any{"badge": "Opening sale: 50% off", "title": "Supermarket daily fresh grocery",
					"subtitle":     "A new way of grocery shopping, with convenient home delivery.",
					"button_label": "Shop now", "button_link": "/search?q=", "align": "left", "height": "lg"})),
			n("categories-section", "section", nil,
				heading("categories-title", "Shop popular categories", 3),
				n("categories", "collection-list", map[string]any{"source": "all", "limit": 12}),
			),
			n("offers-section", "section", nil,
				n("offers", "columns", map[string]any{"stack_on": "tablet"},
					offer("offer-produce", "Fruits & vegetables", "Get up to 30% off.", "secondary"),
					offer("offer-bakery", "Freshly baked buns", "Get up to 25% off.", "secondary"),
				)),
			n("popular-section", "section", nil,
				heading("popular-title", "Popular products", 3),
				freshGrid("popular-grid", 10, 5, "compact"),
			),
			n("best-section", "section", nil,
				heading("best-title", "Daily best sells", 3),
				n("best-cols", "columns", map[string]any{"stack_on": "tablet"},
					darkOffer("best-promo", "100% organic coffee beans.", "Get the best deal before close."),
					n("best-products", "column", map[string]any{"span": 3}, freshGrid("best-grid", 3, 3, "full")),
				),
			),
			n("promises-section", "section", nil,
				n("promises", "columns", map[string]any{"stack_on": "tablet"},
					promise("promise-returns", "refresh", "Easy returns", "Not happy with a product? Return it at the door and get a refund within hours."),
					promise("promise-range", "box", "Wide assortment", "Thousands of products: fresh food, bakery, drinks, household and more."),
					promise("promise-prices", "gift", "Best prices & offers", "Cheaper than your local supermarket, with new deals every week."),
					promise("promise-fast", "clock", "Fast delivery", "Your order at your door within hours."),
				)),
		)
	case "collection", "search":
		return mercadoTemplate(key)
	}
	return defaultTemplate(key)
}

func freshHeader() Node {
	return root(
		n("header-top", "section", nil,
			n("header-top-cols", "columns", map[string]any{"stack_on": "never"},
				n("header-top-text", "column", map[string]any{"span": 3}, paragraph("deals", "Super value deals - Save more with coupons")),
				n("header-actions", "column", nil,
					n("languages", "language-switcher", map[string]any{"display": "codes"}),
					n("wishlist-link", "navigation-menu", map[string]any{"orientation": "horizontal",
						"items": []any{map[string]any{"label": "Wishlist", "link": "/wishlist"}}}),
					n("account", "account-link", map[string]any{"login_label": "", "account_label": ""}),
					n("cart", "cart-link", map[string]any{"label": "", "show_count": true}),
				),
			),
		),
		n("header-main", "section", nil,
			n("header-cols", "columns", map[string]any{"stack_on": "tablet"},
				n("header-brand", "column", nil, n("logo", "site-logo", map[string]any{"size": "md"})),
				n("header-search", "column", map[string]any{"span": 2}, n("search", "search-box", map[string]any{"placeholder": "Type to search..."})),
				n("header-menu", "column", map[string]any{"span": 3},
					n("departments", "navigation-menu", map[string]any{"orientation": "horizontal", "items": []any{
						map[string]any{"label": "All departments", "link": "/search?q="},
						map[string]any{"label": "Home", "link": "/"},
						map[string]any{"label": "Shop", "link": "/search?q="},
						map[string]any{"label": "About", "link": "/pages/about"},
						map[string]any{"label": "Contact", "link": "/pages/contact"},
						map[string]any{"label": "Account", "link": "/account"},
					}})),
			),
		),
	)
}

func freshFooter() Node {
	return root(n("footer-main", "section", nil,
		n("footer-cols", "columns", map[string]any{"stack_on": "tablet"},
			n("footer-about", "column", nil,
				n("footer-logo", "site-logo", map[string]any{"size": "lg"}),
				paragraph("footer-about-text", "Fresh groceries and everyday essentials, delivered to your door. We deliver more than you expect."),
			),
			n("footer-categories", "column", nil,
				heading("footer-categories-title", "All categories", 3),
				n("footer-categories-menu", "navigation-menu", map[string]any{"orientation": "vertical", "items": []any{
					map[string]any{"label": "Dairy, bread & eggs", "link": "/search?q="},
					map[string]any{"label": "Snacks & munchies", "link": "/search?q="},
					map[string]any{"label": "Fruits & vegetables", "link": "/search?q="},
					map[string]any{"label": "Cold drinks & juices", "link": "/search?q="},
					map[string]any{"label": "Bakery & biscuits", "link": "/search?q="},
				}}),
			),
			n("footer-shoppers", "column", nil,
				heading("footer-shoppers-title", "For shoppers", 3),
				n("footer-shoppers-menu", "navigation-menu", map[string]any{"orientation": "vertical", "items": []any{
					map[string]any{"label": "My account", "link": "/account"},
					map[string]any{"label": "Wishlist", "link": "/wishlist"},
					map[string]any{"label": "Shipping", "link": "/pages/shipping"},
					map[string]any{"label": "Returns", "link": "/pages/returns"},
					map[string]any{"label": "FAQ", "link": "/pages/faq"},
				}}),
			),
			n("footer-company", "column", nil,
				heading("footer-company-title", "Get to know us", 3),
				n("footer-company-menu", "navigation-menu", map[string]any{"orientation": "vertical", "items": []any{
					map[string]any{"label": "About", "link": "/pages/about"},
					map[string]any{"label": "Privacy policy", "link": "/pages/privacy"},
					map[string]any{"label": "Terms & conditions", "link": "/pages/terms"},
				}}),
				n("newsletter", "newsletter-signup", map[string]any{"title": "", "text": "", "button_label": "", "consent_label": ""}),
				n("social", "social-links", map[string]any{"size": "md", "show_labels": false, "items": []any{
					map[string]any{"network": "instagram", "link": "https://instagram.com/"},
					map[string]any{"network": "facebook", "link": "https://facebook.com/"},
					map[string]any{"network": "x", "link": "https://x.com/"},
					map[string]any{"network": "whatsapp", "link": "https://wa.me/"},
				}}),
			),
		),
	))
}
