package pms

import (
	"fmt"
	"github.com/adirkuhn/commerceos/domain/errs"
	"strconv"
)

// ErrNotWeightEmbeddedBarcode means the given code isn't a 13-digit,
// GS1-reserved-prefix (20-29) variable-weight barcode at all -- the normal
// exact-match barcode/alias/SKUCode lookups should be tried instead.
var ErrNotWeightEmbeddedBarcode = errs.New("NOT_WEIGHT_EMBEDDED_BARCODE", "not a weight-embedded barcode")

// ErrInvalidCheckDigit means the code has the right shape (13 digits,
// prefix 20-29) but fails the standard EAN-13 checksum -- a corrupted or
// mis-scanned read, not a barcode to resolve against.
var ErrInvalidCheckDigit = errs.New("BARCODE_INVALID_CHECK_DIGIT", "invalid EAN-13 check digit")

// ErrInvalidScaleCode guards EncodeWeightEmbeddedBarcode: scaleCode must be
// exactly 7 digits (2-digit GS1 prefix 20-29 + 5-digit merchant scale code),
// the same shape DecodeWeightEmbeddedBarcode always produces.
var ErrInvalidScaleCode = errs.New("SCALE_CODE_INVALID", "scale code must be 7 digits with a GS1 prefix of 20-29")

// ErrWeightOutOfRange guards EncodeWeightEmbeddedBarcode: the format has
// exactly 5 digits for weight, so 0-99999 grams (0-99.999kg) is the whole
// representable range.
var ErrWeightOutOfRange = errs.New("WEIGHT_OUT_OF_RANGE", "weight must be between 0 and 99999 grams to fit a weight-embedded barcode")

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
	return computeEAN13CheckDigit(code[:12]) == int(code[12]-'0')
}

// computeEAN13CheckDigit is the one place the EAN-13 mod-10 checksum is
// computed -- shared by hasValidEAN13CheckDigit (verifying a scanned code)
// and EncodeWeightEmbeddedBarcode (generating one), so the two can never
// drift apart the way a second hand-rolled copy risks.
func computeEAN13CheckDigit(first12 string) int {
	sum := 0
	for i := 0; i < 12; i++ {
		d := int(first12[i] - '0')
		if i%2 == 0 {
			sum += d
		} else {
			sum += d * 3
		}
	}
	return (10 - sum%10) % 10
}

// EncodeWeightEmbeddedBarcode builds a GS1 "prefix 2x" variable-weight
// EAN-13 barcode from a 7-digit ScaleCode (2-digit GS1 prefix + 5-digit
// merchant-programmed scale code, see DecodedWeightEmbeddedBarcode) and a
// weight in grams -- the inverse of DecodeWeightEmbeddedBarcode. Used when
// this codebase generates a barcode itself (e.g. bulk-portioning several
// packages off a weight pool at once, see
// application/pms.PackWeightPortionsUseCase) rather than reading one a
// physical scale already printed. Reuses computeEAN13CheckDigit's exact
// formula, so anything this produces is guaranteed to decode back through
// DecodeWeightEmbeddedBarcode -- the two can never disagree about what's a
// valid code.
func EncodeWeightEmbeddedBarcode(scaleCode string, weightGrams int64) (string, error) {
	if len(scaleCode) != 7 {
		return "", ErrInvalidScaleCode
	}
	for _, c := range scaleCode {
		if c < '0' || c > '9' {
			return "", ErrInvalidScaleCode
		}
	}
	prefix, _ := strconv.Atoi(scaleCode[0:2])
	if prefix < 20 || prefix > 29 {
		return "", ErrInvalidScaleCode
	}
	if weightGrams < 0 || weightGrams > 99999 {
		return "", ErrWeightOutOfRange
	}
	first12 := fmt.Sprintf("%s%05d", scaleCode, weightGrams)
	check := computeEAN13CheckDigit(first12)
	return fmt.Sprintf("%s%d", first12, check), nil
}
