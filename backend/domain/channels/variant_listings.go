package channels

// VariantListingRef is what's needed to take one variant off one channel
// after its listing row is gone: the channel's own id for it and, for a
// Woo variation, its parent product's id (variations live under the
// parent's endpoint).
type VariantListingRef struct {
	ListingID        string
	ConnectionID     string
	Provider         string
	ExternalID       string
	ParentExternalID string // non-empty only for a Woo variation
}

// ListingsOfVariant picks variantID's listings out of all of its product's
// listings, resolving each variation's parent from the sibling
// product-level row (VariantID nil) on the same connection.
func ListingsOfVariant(all []ProductChannelListing, variantID string) []VariantListingRef {
	var out []VariantListingRef
	for _, l := range all {
		if l.VariantID == nil || *l.VariantID != variantID {
			continue
		}
		ref := VariantListingRef{ListingID: l.ID, ConnectionID: l.ConnectionID, Provider: l.Provider, ExternalID: l.ExternalID}
		if l.IsVariation {
			for _, p := range all {
				if p.VariantID == nil && p.ConnectionID == l.ConnectionID {
					ref.ParentExternalID = p.ExternalID
					break
				}
			}
		}
		out = append(out, ref)
	}
	return out
}
