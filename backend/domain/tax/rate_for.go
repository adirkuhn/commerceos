package tax

import "strings"

// RateFor is the rate of taxClassID in country -- a nil class means the
// tenant's default class. ok is false when no class or no rate for the
// country is configured.
func RateFor(classes []TaxClass, rates []TaxRate, taxClassID *string, country string) (basisPoints int32, ok bool) {
	classID := ""
	if taxClassID != nil {
		classID = *taxClassID
	} else {
		for _, c := range classes {
			if c.IsDefault {
				classID = c.ID
				break
			}
		}
	}
	if classID == "" {
		return 0, false
	}
	for _, r := range rates {
		if r.TaxClassID == classID && strings.EqualFold(r.Country, country) {
			return r.RateBasisPoints, true
		}
	}
	return 0, false
}
