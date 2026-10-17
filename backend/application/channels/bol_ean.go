package channels

import (
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

// bolEAN is the EAN Bol lists sku under: the override's EAN when set, else
// the SKU's barcode, always as the 13 digits Bol accepts (a 12-digit UPC-A
// gets its leading zero). The stored barcode itself is left as it is.
func bolEAN(sku pmsDomain.SKU, override *channelsDomain.BolProductOverride) (string, error) {
	code := sku.Barcode
	if override != nil && override.EANOverride != nil && *override.EANOverride != "" {
		code = *override.EANOverride
	}
	if code == "" {
		return "", channelsDomain.ErrBolMissingEAN
	}
	ean, ok := pmsDomain.ToGTIN13(code)
	if !ok {
		return "", channelsDomain.ErrBolInvalidEAN
	}
	return ean, nil
}
