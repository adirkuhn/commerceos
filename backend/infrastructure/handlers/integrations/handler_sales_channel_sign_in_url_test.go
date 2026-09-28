package integrations_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	genintegrations "github.com/adirkuhn/commerceos/api/integrations"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
)

func TestUpdateSalesChannelHandler_passesSignInURLThroughAndReturnsIt(t *testing.T) {
	page := "https://shop.example.com/verify"
	update := &stubUpdateSalesChannel{out: &channelsDomain.SalesChannel{ID: "chan-1", Name: "Shop", EnrollmentMode: channelsDomain.EnrollmentModeAll, SignInURL: &page}}
	body := genintegrations.UpdateSalesChannelRequest{Name: "Shop", EnrollmentMode: genintegrations.EnrollmentMode(channelsDomain.EnrollmentModeAll), Active: true, SignInUrl: &page}

	w := doRequest(newStorefrontModeRouter(&stubCreateSalesChannel{}, update), http.MethodPatch, "/api/v1/integrations/sales-channels/chan-1", body)

	require.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, update.in.SignInURL)
	assert.Equal(t, page, *update.in.SignInURL)
	assert.Contains(t, w.Body.String(), `"sign_in_url":"https://shop.example.com/verify"`)
}

func TestUpdateSalesChannelHandler_omittedSignInURLPassesNil(t *testing.T) {
	update := &stubUpdateSalesChannel{out: &channelsDomain.SalesChannel{ID: "chan-1", Name: "Shop", EnrollmentMode: channelsDomain.EnrollmentModeAll}}
	body := genintegrations.UpdateSalesChannelRequest{Name: "Shop", EnrollmentMode: genintegrations.EnrollmentMode(channelsDomain.EnrollmentModeAll), Active: true}

	w := doRequest(newStorefrontModeRouter(&stubCreateSalesChannel{}, update), http.MethodPatch, "/api/v1/integrations/sales-channels/chan-1", body)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Nil(t, update.in.SignInURL)
}

func TestCreateSalesChannelHandler_unsafeSignInURLIs422(t *testing.T) {
	create := &stubCreateSalesChannel{err: channelsDomain.ErrInvalidSignInURL}
	bad := "javascript:alert(1)"
	body := genintegrations.CreateSalesChannelRequest{
		Name: "Shop", Type: genintegrations.ChannelType(channelsDomain.ChannelTypeInternal),
		EnrollmentMode: genintegrations.EnrollmentMode(channelsDomain.EnrollmentModeAll), SignInUrl: &bad,
	}

	w := doRequest(newStorefrontModeRouter(create, &stubUpdateSalesChannel{}), http.MethodPost, "/api/v1/integrations/sales-channels", body)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}
