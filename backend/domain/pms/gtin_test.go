package pms

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToGTIN13(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
		ok   bool
	}{
		{"EAN-13 kept as is", "8710398527806", "8710398527806", true},
		{"UPC-A gets a leading zero", "875754010311", "0875754010311", true},
		{"surrounding spaces ignored", " 875754010311 ", "0875754010311", true},
		{"GTIN-14 with leading zero drops it", "00875754010311", "0875754010311", true},
		{"GTIN-14 with packaging digit is not an EAN-13", "10875754010318", "", false},
		{"bad check digit", "875754010312", "", false},
		{"bad EAN-13 check digit", "8710398527807", "", false},
		{"letters", "87575401031A", "", false},
		{"too short", "12345", "", false},
		{"empty", "", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := ToGTIN13(tc.in)
			assert.Equal(t, tc.ok, ok)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestUPCAFromGTIN13(t *testing.T) {
	upc, ok := UPCAFromGTIN13("0875754010311")
	assert.True(t, ok)
	assert.Equal(t, "875754010311", upc)

	_, ok = UPCAFromGTIN13("8710398527806")
	assert.False(t, ok, "an EAN-13 not starting with 0 has no UPC-A form")
	_, ok = UPCAFromGTIN13("087575401031")
	assert.False(t, ok, "not 13 digits")
}
