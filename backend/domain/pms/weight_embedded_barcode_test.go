package pms

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeWeightEmbeddedBarcode_RealScaleSamples(t *testing.T) {
	// Real barcodes off the user's scale-printed labels -- calabresa and
	// kit feijoada share a ScaleCode per product across different package
	// weights; dry meat likewise. Each is a valid GS1 "prefix 2x"
	// variable-weight EAN-13: prefix(2) + scale_code(5) + weight_grams(5)
	// + standard EAN-13 check digit(1).
	tests := []struct {
		barcode   string
		wantScale string
		wantGrams int64
	}{
		{"2900633003332", "00633", 333},
		{"2900633003288", "00633", 328},
		{"2908885004077", "08885", 407},
		{"2908885004053", "08885", 405},
		{"2994721004622", "94721", 462},
		{"2994721004943", "94721", 494},
	}
	for _, tt := range tests {
		t.Run(tt.barcode, func(t *testing.T) {
			got, err := DecodeWeightEmbeddedBarcode(tt.barcode)
			require.NoError(t, err)
			assert.Equal(t, tt.wantScale, got.ScaleCode)
			assert.Equal(t, tt.wantGrams, got.WeightGrams)
		})
	}
}

func TestDecodeWeightEmbeddedBarcode_RejectsWrongPrefix(t *testing.T) {
	// Prefix 30 is outside the GS1 20-29 reserved internal-circulation
	// block -- an ordinary retail EAN-13, not a scale barcode, must fall
	// through to normal exact-barcode lookup instead of being misread.
	_, err := DecodeWeightEmbeddedBarcode("3000633003332")
	assert.ErrorIs(t, err, ErrNotWeightEmbeddedBarcode)
}

func TestDecodeWeightEmbeddedBarcode_AcceptsWholePrefixBlock(t *testing.T) {
	// 20-29 is the whole GS1-reserved internal-use block, not just the 29
	// this store happens to be assigned -- covers another scale/vendor
	// using e.g. 23 without code changes.
	for prefix := 20; prefix <= 29; prefix++ {
		barcode := recomputeCheckDigit(itoaPrefix(prefix) + "0063300333")
		t.Run(barcode, func(t *testing.T) {
			got, err := DecodeWeightEmbeddedBarcode(barcode)
			require.NoError(t, err)
			assert.Equal(t, "00633", got.ScaleCode)
			assert.Equal(t, int64(333), got.WeightGrams)
		})
	}
}

func TestDecodeWeightEmbeddedBarcode_RejectsBadCheckDigit(t *testing.T) {
	// Last digit corrupted (2 -> 9 in the last real sample above) -- must
	// be rejected rather than silently trusting a mis-scanned weight.
	_, err := DecodeWeightEmbeddedBarcode("2900633003339")
	assert.ErrorIs(t, err, ErrInvalidCheckDigit)
}

func TestDecodeWeightEmbeddedBarcode_RejectsWrongLength(t *testing.T) {
	for _, barcode := range []string{"", "290063300333", "29006330033322", "abcdefghijklm"} {
		t.Run(barcode, func(t *testing.T) {
			_, err := DecodeWeightEmbeddedBarcode(barcode)
			assert.True(t, errors.Is(err, ErrNotWeightEmbeddedBarcode))
		})
	}
}

// itoaPrefix/recomputeCheckDigit are tiny test-only helpers -- the
// production check-digit math lives in weight_embedded_barcode.go and is
// exercised directly by the real-sample tests above; these just let the
// prefix-range test build valid fixtures without hardcoding all 10 check
// digits by hand.
func itoaPrefix(n int) string {
	return string(rune('0'+n/10)) + string(rune('0'+n%10))
}

func recomputeCheckDigit(first12 string) string {
	sum := 0
	for i, c := range first12 {
		d := int(c - '0')
		if i%2 == 0 {
			sum += d
		} else {
			sum += d * 3
		}
	}
	check := (10 - sum%10) % 10
	return first12 + string(rune('0'+check))
}
