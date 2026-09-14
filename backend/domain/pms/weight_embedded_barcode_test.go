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
	// + standard EAN-13 check digit(1). ScaleCode is prefix+scale_code (7
	// digits) -- stored and matched exactly as read off the scale's own
	// config screen, no stripping.
	tests := []struct {
		barcode   string
		wantScale string
		wantGrams int64
	}{
		{"2900633003332", "2900633", 333},
		{"2900633003288", "2900633", 328},
		{"2908885004077", "2908885", 407},
		{"2908885004053", "2908885", 405},
		{"2994721004622", "2994721", 462},
		{"2994721004943", "2994721", 494},
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
	// using e.g. 23 without code changes. Fixtures are built via
	// EncodeWeightEmbeddedBarcode itself rather than a hand-rolled checksum
	// -- dogfoods the real encoder and proves it round-trips through Decode.
	for prefix := 20; prefix <= 29; prefix++ {
		prefixStr := itoaPrefix(prefix)
		barcode, err := EncodeWeightEmbeddedBarcode(prefixStr+"00633", 333)
		require.NoError(t, err)
		t.Run(barcode, func(t *testing.T) {
			got, err := DecodeWeightEmbeddedBarcode(barcode)
			require.NoError(t, err)
			assert.Equal(t, prefixStr+"00633", got.ScaleCode)
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

// itoaPrefix is a tiny test-only helper letting the prefix-range test build
// fixtures without hardcoding "20".."29" as string literals.
func itoaPrefix(n int) string {
	return string(rune('0'+n/10)) + string(rune('0'+n%10))
}

func TestEncodeWeightEmbeddedBarcode_MatchesRealScaleSamples(t *testing.T) {
	// Same fixtures as the decode test above, run in reverse -- proves the
	// encoder produces byte-for-byte the same barcode a real scale printed,
	// not just "a barcode that happens to decode back correctly."
	tests := []struct {
		scaleCode string
		grams     int64
		want      string
	}{
		{"2900633", 333, "2900633003332"},
		{"2900633", 328, "2900633003288"},
		{"2908885", 407, "2908885004077"},
		{"2994721", 462, "2994721004622"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got, err := EncodeWeightEmbeddedBarcode(tt.scaleCode, tt.grams)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestEncodeWeightEmbeddedBarcode_RoundTripsThroughDecode(t *testing.T) {
	for _, grams := range []int64{0, 1, 333, 9999, 99999} {
		barcode, err := EncodeWeightEmbeddedBarcode("2994721", grams)
		require.NoError(t, err)
		decoded, err := DecodeWeightEmbeddedBarcode(barcode)
		require.NoError(t, err)
		assert.Equal(t, "2994721", decoded.ScaleCode)
		assert.Equal(t, grams, decoded.WeightGrams)
	}
}

func TestEncodeWeightEmbeddedBarcode_RejectsBadScaleCode(t *testing.T) {
	for _, sc := range []string{"", "299472", "29947210", "299472a", "1994721"} {
		t.Run(sc, func(t *testing.T) {
			_, err := EncodeWeightEmbeddedBarcode(sc, 100)
			assert.ErrorIs(t, err, ErrInvalidScaleCode)
		})
	}
}

func TestEncodeWeightEmbeddedBarcode_RejectsWeightOutOfRange(t *testing.T) {
	_, err := EncodeWeightEmbeddedBarcode("2994721", -1)
	assert.ErrorIs(t, err, ErrWeightOutOfRange)

	_, err = EncodeWeightEmbeddedBarcode("2994721", 100000)
	assert.ErrorIs(t, err, ErrWeightOutOfRange)
}
