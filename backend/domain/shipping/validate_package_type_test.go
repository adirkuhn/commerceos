package shipping_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/adirkuhn/commerceos/domain/shipping"
)

// TestValidatePackageTypeForCarrier_postNL_onlyPackageAndMailboxAllowed pins
// a real MyParcel 422 (error 3707, "Pakkettype niet ondersteund"): PostNL
// (MyParcel's own-brand "SendMyParcel" default carrier) only accepts
// package_type 1 (Package) and 2 (Mailbox package), per MyParcel's own docs.
func TestValidatePackageTypeForCarrier_postNL_onlyPackageAndMailboxAllowed(t *testing.T) {
	assert.NoError(t, shipping.ValidatePackageTypeForCarrier(shipping.CarrierPostNL, shipping.PackageTypePackage))
	assert.NoError(t, shipping.ValidatePackageTypeForCarrier(shipping.CarrierPostNL, shipping.PackageTypeMailboxPackage))

	unsupported := []shipping.PackageType{
		shipping.PackageTypeDigitalStamp, shipping.PackageTypePallet,
		shipping.PackageTypeSmallPackage, shipping.PackageTypeEnvelope,
	}
	for _, pt := range unsupported {
		err := shipping.ValidatePackageTypeForCarrier(shipping.CarrierPostNL, pt)
		assert.True(t, errors.Is(err, shipping.ErrUnsupportedPackageTypeForCarrier), "package type %d must be rejected for PostNL", pt)
	}
}

// TestValidatePackageTypeForCarrier_unlistedCarrier_isUnrestricted covers a
// carrier with no confirmed MyParcel capability restriction -- must not be
// guessed, so every package_type passes.
func TestValidatePackageTypeForCarrier_unlistedCarrier_isUnrestricted(t *testing.T) {
	allTypes := []shipping.PackageType{
		shipping.PackageTypePackage, shipping.PackageTypeMailboxPackage, shipping.PackageTypeDigitalStamp,
		shipping.PackageTypePallet, shipping.PackageTypeSmallPackage, shipping.PackageTypeEnvelope,
	}
	for _, pt := range allTypes {
		assert.NoError(t, shipping.ValidatePackageTypeForCarrier(shipping.CarrierBpost, pt))
	}
}
