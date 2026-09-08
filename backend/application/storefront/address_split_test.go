package storefront

import "testing"

func TestSplitStreetAndNumber(t *testing.T) {
	cases := []struct {
		name     string
		address1 string
		address2 string
		street   string
		number   string
		suffix   string
	}{
		{name: "splits a trailing number", address1: "Damrak 1", street: "Damrak", number: "1"},
		{name: "splits a trailing number with a letter suffix", address1: "Damrak 1A", street: "Damrak", number: "1", suffix: "A"},
		{name: "falls back to the whole string as street when there is no trailing number and no address2", address1: "Damrak", street: "Damrak"},
		{name: "falls back to address2 when it is just a number", address1: "Fossa Iberica", address2: "9", street: "Fossa Iberica", number: "9"},
		{name: "falls back to address2 including a letter suffix", address1: "Fossa Iberica", address2: "9A", street: "Fossa Iberica", number: "9", suffix: "A"},
		{name: "ignores address2 when it is not just a number", address1: "Fossa Iberica", address2: "Apartment 3", street: "Fossa Iberica"},
		{name: "prefers a number already found in address1 over address2", address1: "Damrak 1", address2: "9", street: "Damrak", number: "1"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			street, number, suffix := splitStreetAndNumber(tc.address1, tc.address2)
			if street != tc.street || number != tc.number || suffix != tc.suffix {
				t.Errorf("splitStreetAndNumber(%q, %q) = (%q, %q, %q), want (%q, %q, %q)",
					tc.address1, tc.address2, street, number, suffix, tc.street, tc.number, tc.suffix)
			}
		})
	}
}
