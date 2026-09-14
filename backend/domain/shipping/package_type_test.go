package shipping_test

import (
	"testing"

	"github.com/adirkuhn/commerceos/domain/shipping"
	"github.com/stretchr/testify/assert"
)

// TestPackageType_MyParcelIDs pins this codebase's PackageType consts to
// MyParcel's own shipment_options.package_type enum -- id 3 ("letter") is
// deprecated/unused by MyParcel and intentionally has no const here, same
// as before this test. A wrong id here silently books the wrong box type on
// a real MyParcel shipment, so these are asserted individually rather than
// left as implicit const values.
func TestPackageType_MyParcelIDs(t *testing.T) {
	assert.Equal(t, shipping.PackageType(1), shipping.PackageTypePackage)
	assert.Equal(t, shipping.PackageType(2), shipping.PackageTypeMailboxPackage)
	assert.Equal(t, shipping.PackageType(4), shipping.PackageTypeDigitalStamp)
	assert.Equal(t, shipping.PackageType(5), shipping.PackageTypePallet)
	assert.Equal(t, shipping.PackageType(6), shipping.PackageTypeSmallPackage)
	assert.Equal(t, shipping.PackageType(7), shipping.PackageTypeEnvelope)
}
