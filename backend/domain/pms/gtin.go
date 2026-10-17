package pms

import "strings"

// ToGTIN13 returns code as a 13-digit GTIN (EAN-13), the only form
// marketplaces like Bol accept. A 12-digit UPC-A is the same number with a
// leading zero, and a 14-digit GTIN starting with zero is the same number
// with one extra; any other length, a non-digit or a wrong check digit is
// not an EAN-13 (ok=false).
func ToGTIN13(code string) (gtin string, ok bool) {
	code = strings.TrimSpace(code)
	switch {
	case len(code) == 12:
		code = "0" + code
	case len(code) == 14 && code[0] == '0':
		code = code[1:]
	}
	if len(code) != 13 {
		return "", false
	}
	for i := 0; i < len(code); i++ {
		if code[i] < '0' || code[i] > '9' {
			return "", false
		}
	}
	if !hasValidEAN13CheckDigit(code) {
		return "", false
	}
	return code, true
}

// UPCAFromGTIN13 is ToGTIN13's reverse for a UPC-A: a 13-digit code
// starting with zero is the same number as its last 12 digits. Used to
// match a marketplace's EAN against a barcode stored as a UPC-A.
func UPCAFromGTIN13(ean string) (string, bool) {
	if len(ean) != 13 || ean[0] != '0' {
		return "", false
	}
	return ean[1:], true
}
