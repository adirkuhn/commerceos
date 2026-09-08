package reviews

import "testing"

func TestValidRating(t *testing.T) {
	cases := []struct {
		rating int
		want   bool
	}{
		{0, false},
		{1, true},
		{3, true},
		{5, true},
		{6, false},
		{-1, false},
	}
	for _, tc := range cases {
		r := Review{Rating: tc.rating}
		if got := r.ValidRating(); got != tc.want {
			t.Errorf("ValidRating(%d) = %v, want %v", tc.rating, got, tc.want)
		}
	}
}
