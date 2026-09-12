package pms

import (
	"errors"
	"strconv"
)

// ErrNotWeightEmbeddedBarcode means the given code isn't a 13-digit,
// GS1-reserved-prefix (20-29) variable-weight barcode at all -- the normal
// exact-match barcode/alias/SKUCode lookups should be tried instead.
var ErrNotWeightEmbeddedBarcode = errors.New("not a weight-embedded barcode")

// ErrInvalidCheckDigit means the code has the right shape (13 digits,
// prefix 20-29) but fails the standard EAN-13 checksum -- a corrupted or
// mis-scanned read, not a barcode to resolve against.
var ErrInvalidCheckDigit = errors.New("invalid EAN-13 check digit")

// DecodedWeightEmbeddedBarcode is what a GS1 "prefix 2x" variable-weight
// EAN-13 barcode encodes: a 7-digit scale code (see SKU.ScaleCode) --
// the 2-digit GS1 prefix plus the 5-digit code the tenant programs into
// their scale per product, stored and matched together exactly as printed
// on the scale's own config screen -- and the actual weight of the
// specific package that was printed and scanned.
type DecodedWeightEmbeddedBarcode struct {
	ScaleCode   string
	WeightGrams int64
}

// DecodeWeightEmbeddedBarcode parses a scanned/typed code as a GS1
// "prefix 2x" variable-weight EAN-13 barcode: 2-digit prefix (20-29, the
// GS1-reserved internal-circulation block) + 5-digit scale code + 5-digit
// weight in grams + 1-digit standard EAN-13 check digit. This is the
// barcode format printed by butcher/deli scales -- every package of the
// same product gets a different full barcode (the weight varies), so it
// can never be matched by SKU.Barcode/BarcodeAlias exact lookup; callers
// resolve the extracted ScaleCode (prefix + scale code, 7 digits -- see
// DecodedWeightEmbeddedBarcode) against SKURepository.FindByScaleCode
// instead. Returns ErrNotWeightEmbeddedBarcode when the code doesn't have
// this shape at all (falls through to normal lookup), or
// ErrInvalidCheckDigit when the shape matches but the checksum doesn't
// (corrupted/mis-scanned read -- must not resolve to a possibly-wrong
// product/weight).
func DecodeWeightEmbeddedBarcode(code string) (*DecodedWeightEmbeddedBarcode, error) {
	if len(code) != 13 {
		return nil, ErrNotWeightEmbeddedBarcode
	}
	for _, c := range code {
		if c < '0' || c > '9' {
			return nil, ErrNotWeightEmbeddedBarcode
		}
	}
	prefix, _ := strconv.Atoi(code[0:2])
	if prefix < 20 || prefix > 29 {
		return nil, ErrNotWeightEmbeddedBarcode
	}
	if !hasValidEAN13CheckDigit(code) {
		return nil, ErrInvalidCheckDigit
	}
	weightGrams, _ := strconv.ParseInt(code[7:12], 10, 64)
	return &DecodedWeightEmbeddedBarcode{
		ScaleCode:   code[0:7],
		WeightGrams: weightGrams,
	}, nil
}

// hasValidEAN13CheckDigit applies the standard EAN-13 mod-10 checksum
// (odd positions x1, even positions x3, 1-indexed from the left) to the
// first 12 digits and compares against the 13th.
func hasValidEAN13CheckDigit(code string) bool {
	sum := 0
	for i := 0; i < 12; i++ {
		d := int(code[i] - '0')
		if i%2 == 0 {
			sum += d
		} else {
			sum += d * 3
		}
	}
	check := (10 - sum%10) % 10
	return check == int(code[12]-'0')
}
