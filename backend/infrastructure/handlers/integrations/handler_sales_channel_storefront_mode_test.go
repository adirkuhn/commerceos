package integrations_test

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	genintegrations "github.com/adirkuhn/commerceos/api/integrations"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationshandler "github.com/adirkuhn/commerceos/infrastructure/handlers/integrations"
)

func newStorefrontModeRouter(create *stubCreateSalesChannel, update *stubUpdateSalesChannel) *gin.Engine {
	h := newSalesChannelTestHandler(&stubListSalesChannels{}, create, update, &stubDeleteSalesChannel{}, &stubListSalesChannelProducts{}, &stubSetProductChannelOverride{})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)
	return r
}

func TestCreateSalesChannelHandler_passesStorefrontModeThroughAndReturnsIt(t *testing.T) {
	create := &stubCreateSalesChannel{out: &channelsDomain.SalesChannel{
		ID: "chan-1", Name: "Shop", Type: channelsDomain.ChannelTypeInternal, EnrollmentMode: channelsDomain.EnrollmentModeAll,
		StorefrontMode: channelsDomain.StorefrontModeTheme,
	}}
	mode := genintegrations.Theme
	body := genintegrations.CreateSalesChannelRequest{
		Name: "Shop", Type: genintegrations.ChannelType(channelsDomain.ChannelTypeInternal),
		EnrollmentMode: genintegrations.EnrollmentMode(channelsDomain.EnrollmentModeAll), StorefrontMode: &mode,
	}

	w := doRequest(newStorefrontModeRouter(create, &stubUpdateSalesChannel{}), http.MethodPost, "/api/v1/integrations/sales-channels", body)

	require.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, channelsDomain.StorefrontModeTheme, create.in.StorefrontMode)
	assert.Contains(t, w.Body.String(), `"storefront_mode":"theme"`)
}

func TestCreateSalesChannelHandler_themedStorefrontOnExternalChannelIs422(t *testing.T) {
	create := &stubCreateSalesChannel{err: channelsDomain.ErrThemedStorefrontRequiresInternal}
	mode := genintegrations.Theme
	body := genintegrations.CreateSalesChannelRequest{
		Name: "Woo", Type: genintegrations.ChannelType(channelsDomain.ChannelTypeWooCommerce),
		EnrollmentMode: genintegrations.EnrollmentMode(channelsDomain.EnrollmentModeAll), StorefrontMode: &mode,
	}

	w := doRequest(newStorefrontModeRouter(create, &stubUpdateSalesChannel{}), http.MethodPost, "/api/v1/integrations/sales-channels", body)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestUpdateSalesChannelHandler_omittedStorefrontModePassesEmpty(t *testing.T) {
	update := &stubUpdateSalesChannel{out: &channelsDomain.SalesChannel{ID: "chan-1", Name: "Shop", EnrollmentMode: channelsDomain.EnrollmentModeAll, StorefrontMode: channelsDomain.StorefrontModeTheme}}
	body := genintegrations.UpdateSalesChannelRequest{Name: "Shop", EnrollmentMode: genintegrations.EnrollmentMode(channelsDomain.EnrollmentModeAll), Active: true}

	w := doRequest(newStorefrontModeRouter(&stubCreateSalesChannel{}, update), http.MethodPatch, "/api/v1/integrations/sales-channels/chan-1", body)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, channelsDomain.StorefrontMode(""), update.in.StorefrontMode)
}

func TestUpdateSalesChannelHandler_passesStorefrontModeThrough(t *testing.T) {
	update := &stubUpdateSalesChannel{out: &channelsDomain.SalesChannel{ID: "chan-1", Name: "Shop", EnrollmentMode: channelsDomain.EnrollmentModeAll, StorefrontMode: channelsDomain.StorefrontModeHeadless}}
	mode := genintegrations.Headless
	body := genintegrations.UpdateSalesChannelRequest{Name: "Shop", EnrollmentMode: genintegrations.EnrollmentMode(channelsDomain.EnrollmentModeAll), Active: true, StorefrontMode: &mode}

	w := doRequest(newStorefrontModeRouter(&stubCreateSalesChannel{}, update), http.MethodPatch, "/api/v1/integrations/sales-channels/chan-1", body)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, channelsDomain.StorefrontModeHeadless, update.in.StorefrontMode)
}

func TestUpdateSalesChannelHandler_invalidStorefrontModeIs422(t *testing.T) {
	update := &stubUpdateSalesChannel{err: channelsDomain.ErrInvalidStorefrontMode}
	mode := genintegrations.StorefrontMode("spa")
	body := genintegrations.UpdateSalesChannelRequest{Name: "Shop", EnrollmentMode: genintegrations.EnrollmentMode(channelsDomain.EnrollmentModeAll), Active: true, StorefrontMode: &mode}

	w := doRequest(newStorefrontModeRouter(&stubCreateSalesChannel{}, update), http.MethodPatch, "/api/v1/integrations/sales-channels/chan-1", body)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}
