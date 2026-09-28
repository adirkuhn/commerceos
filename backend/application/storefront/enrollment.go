package storefront

import (
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
)

// enrollment is which products a channel sells, as pms.ProductFilter's
// OR-group (categoryID/tag/includeIDs) plus the always-applied excludes --
// see pms.ProductFilter.IncludeIDs' doc comment for the nil-vs-empty rule.
type enrollment struct {
	categoryID *string
	tag        *string
	includeIDs *[]string
	excludeIDs []string
}

// channelEnrollment translates a channel's EnrollmentMode and its manual
// per-product overrides into that filter shape.
func channelEnrollment(channel *channelsDomain.SalesChannel, overrides []channelsDomain.SalesChannelProductOverride) enrollment {
	includeIDs := []string{}
	var e enrollment
	for _, o := range overrides {
		if o.Included {
			includeIDs = append(includeIDs, o.ProductID)
		} else {
			e.excludeIDs = append(e.excludeIDs, o.ProductID)
		}
	}
	switch channel.EnrollmentMode {
	case channelsDomain.EnrollmentModeAll:
		// Everything active is enrolled; excludes still apply.
	case channelsDomain.EnrollmentModeCategory:
		e.categoryID = channel.EnrollmentCategoryID
		e.includeIDs = &includeIDs
	case channelsDomain.EnrollmentModeTag:
		e.tag = &channel.EnrollmentTag
		e.includeIDs = &includeIDs
	case channelsDomain.EnrollmentModeManual:
		e.includeIDs = &includeIDs
	}
	return e
}
