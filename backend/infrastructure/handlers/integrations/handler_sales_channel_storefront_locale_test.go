package integrations_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	genintegrations "github.com/adirkuhn/commerceos/api/integrations"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
)

func TestCreateSalesChannelHandler_passesStorefrontLocaleAndReturnsIt(t *testing.T) {
	create := &stubCreateSalesChannel{out: &channelsDomain.SalesChannel{
		ID: "chan-1", Name: "Loja", Type: channelsDomain.ChannelTypeInternal, EnrollmentMode: channelsDomain.EnrollmentModeAll,
		StorefrontLocale: channelsDomain.StorefrontLocalePortuguese,
	}}
	locale := genintegrations.PtBR
	body := genintegrations.CreateSalesChannelRequest{
		Name: "Loja", Type: genintegrations.ChannelType(channelsDomain.ChannelTypeInternal),
		EnrollmentMode: genintegrations.EnrollmentMode(channelsDomain.EnrollmentModeAll), StorefrontLocale: &locale,
	}

	w := doRequest(newStorefrontModeRouter(create, &stubUpdateSalesChannel{}), http.MethodPost, "/api/v1/integrations/sales-channels", body)

	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	assert.Equal(t, channelsDomain.StorefrontLocalePortuguese, create.in.StorefrontLocale)
	assert.Contains(t, w.Body.String(), `"storefront_locale":"pt-BR"`)
}

func TestUpdateSalesChannelHandler_omittedLocalePassesEmptyAndUnknownIs422(t *testing.T) {
	update := &stubUpdateSalesChannel{out: &channelsDomain.SalesChannel{ID: "chan-1", Name: "Shop", EnrollmentMode: channelsDomain.EnrollmentModeAll}}
	body := genintegrations.UpdateSalesChannelRequest{Name: "Shop", EnrollmentMode: genintegrations.EnrollmentMode(channelsDomain.EnrollmentModeAll), Active: true}
	w := doRequest(newStorefrontModeRouter(&stubCreateSalesChannel{}, update), http.MethodPatch, "/api/v1/integrations/sales-channels/chan-1", body)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, channelsDomain.StorefrontLocale(""), update.in.StorefrontLocale)

	failing := &stubUpdateSalesChannel{err: channelsDomain.ErrInvalidStorefrontLocale}
	w = doRequest(newStorefrontModeRouter(&stubCreateSalesChannel{}, failing), http.MethodPatch, "/api/v1/integrations/sales-channels/chan-1", body)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}
