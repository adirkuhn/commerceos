package storefront

import "github.com/adirkuhn/commerceos/domain/locale"

// In is the product summary as read in l (translated name).
func (p ProductSummary) In(l locale.Locale) ProductSummary {
	p.Product = p.Product.In(l)
	return p
}

// In is the product as read in l (translated name and description).
func (d ProductDetail) In(l locale.Locale) ProductDetail {
	d.Product = d.Product.In(l)
	return d
}

// In is the collection as read in l (translated name, description, SEO).
func (c StorefrontCollection) In(l locale.Locale) StorefrontCollection {
	c.Collection = c.Collection.In(l)
	return c
}
