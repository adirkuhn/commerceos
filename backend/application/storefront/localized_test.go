package storefront_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	appstorefront "github.com/adirkuhn/commerceos/application/storefront"
	"github.com/adirkuhn/commerceos/domain/collections"
	"github.com/adirkuhn/commerceos/domain/locale"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

func TestShopperFacingTextsIn(t *testing.T) {
	product := pmsDomain.Product{Name: "Pão de queijo", Description: "Quentinho",
		Translations: map[locale.Locale]pmsDomain.ProductTexts{locale.English: {Name: "Cheese bread", Description: "Warm"}}}
	price := int64(250)

	summary := appstorefront.ProductSummary{Product: product, PriceMinorUnits: &price}.In(locale.English)
	assert.Equal(t, "Cheese bread", summary.Name)
	assert.Equal(t, &price, summary.PriceMinorUnits)

	detail := appstorefront.ProductDetail{Product: product, Images: []string{"a.jpg"}}.In(locale.English)
	assert.Equal(t, "Warm", detail.Description)
	assert.Equal(t, []string{"a.jpg"}, detail.Images)

	col := appstorefront.StorefrontCollection{Collection: collections.Collection{Draft: collections.Draft{Name: "Verão",
		Translations: map[locale.Locale]collections.Texts{locale.English: {Name: "Summer"}}}}}
	assert.Equal(t, "Summer", col.In(locale.English).Name)
	assert.Equal(t, "Verão", col.In("").Name)
}
