package storefront

import "testing"

func TestWishlistContains(t *testing.T) {
	w := Wishlist{ProductIDs: []string{"product-1", "product-2"}}

	if !w.Contains("product-1") {
		t.Error("expected Contains(product-1) to be true")
	}
	if w.Contains("product-3") {
		t.Error("expected Contains(product-3) to be false")
	}
}
