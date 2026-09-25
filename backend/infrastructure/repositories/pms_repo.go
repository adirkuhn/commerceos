package repositories

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// fuzzySearchSimilarityThreshold is the pg_trgm similarity() cutoff (0..1,
// higher = stricter) below which a product name doesn't count as a fuzzy
// match at all -- keeps a search for "Widget" from also fuzzy-matching
// "Gadget" just because they share a few trigrams. 0.25 is more permissive
// than pg_trgm's own 0.3 default: this app's product names skew short, and
// a stricter cutoff missed short-name typos ("Widgt" vs "Widget") in
// testing.
const fuzzySearchSimilarityThreshold = 0.25

// stringSlice is a zero-dependency []string <-> jsonb Scanner/Valuer --
// avoids pulling in a Postgres-array driver type for the one column that
// needs it (products.tags).
type stringSlice []string

func (s stringSlice) Value() (driver.Value, error) {
	if len(s) == 0 {
		return "[]", nil
	}
	b, err := json.Marshal([]string(s))
	return string(b), err
}

func (s *stringSlice) Scan(value any) error {
	if value == nil {
		*s = nil
		return nil
	}
	var b []byte
	switch v := value.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return fmt.Errorf("stringSlice.Scan: unsupported type %T", value)
	}
	return json.Unmarshal(b, (*[]string)(s))
}
