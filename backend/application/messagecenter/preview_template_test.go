package messagecenter_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appmc "github.com/adirkuhn/commerceos/application/messagecenter"
	mcDomain "github.com/adirkuhn/commerceos/domain/messagecenter"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func TestPreviewTemplate_rendersWithSampleDataAndTheRealStore(t *testing.T) {
	uc := appmc.NewPreviewTemplateUseCase(newMockGormDB(t, 1), fixedProfile{&settingsDomain.CompanyProfile{TradingName: "Cantinho Brasileiro"}})

	out, err := uc.Execute(context.Background(), appmc.PreviewTemplateInput{
		TriggerSubject: "storefront.order.placed",
		Subject:        "Order {{order_number}}",
		Body:           "<p>{{store_name}}: {{#items}}{{name}} & co;{{/items}}</p>",
		TextBody:       "Hi {{customer_name}}",
	})

	require.NoError(t, err)
	assert.Equal(t, "Order 1001", out.Subject)
	assert.Equal(t, "<p>Cantinho Brasileiro: Pão de queijo 400g & co;Pão de queijo 400g & co;</p>", out.HTML)
	assert.Equal(t, "Hi Ana", out.Text)
}

func TestPreviewTemplate_textFallsBackToBody(t *testing.T) {
	uc := appmc.NewPreviewTemplateUseCase(newMockGormDB(t, 1), fixedProfile{&settingsDomain.CompanyProfile{}})
	out, err := uc.Execute(context.Background(), appmc.PreviewTemplateInput{TriggerSubject: mcDomain.TriggerStockCloseToExpiry, Body: "{{window_days}} days"})
	require.NoError(t, err)
	assert.Equal(t, "30 days", out.Text)
}

func TestPreviewTemplate_unknownTrigger(t *testing.T) {
	uc := appmc.NewPreviewTemplateUseCase(newMockGormDB(t, 0), fixedProfile{})
	_, err := uc.Execute(context.Background(), appmc.PreviewTemplateInput{TriggerSubject: "nope"})
	assert.ErrorIs(t, err, mcDomain.ErrInvalidTriggerSubject)
}

type fakeImageStorage struct {
	gotTenant, gotName string
	gotBytes           []byte
	err                error
}

func (f *fakeImageStorage) Save(_ context.Context, tenantID, filename string, data io.Reader) (string, error) {
	f.gotTenant, f.gotName = tenantID, filename
	f.gotBytes, _ = io.ReadAll(data)
	return "https://cdn/" + filename, f.err
}

var pngBytes = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR")

func TestUploadTemplateImage_savesAnImageAndReturnsItsURL(t *testing.T) {
	storage := &fakeImageStorage{}
	uc := appmc.NewUploadTemplateImageUseCase(storage)

	url, err := uc.Execute(tenant.WithTenantID(context.Background(), "tenant-1"), bytes.NewReader(pngBytes))

	require.NoError(t, err)
	assert.Equal(t, "tenant-1", storage.gotTenant)
	assert.True(t, strings.HasPrefix(storage.gotName, "message-images/") && strings.HasSuffix(storage.gotName, ".png"), storage.gotName)
	assert.Equal(t, pngBytes, storage.gotBytes)
	assert.Equal(t, "https://cdn/"+storage.gotName, url)
}

func TestUploadTemplateImage_rejectsNonImagesAndHugeFiles(t *testing.T) {
	uc := appmc.NewUploadTemplateImageUseCase(&fakeImageStorage{})
	ctx := tenant.WithTenantID(context.Background(), "tenant-1")

	_, err := uc.Execute(ctx, strings.NewReader("<html>not an image</html>"))
	assert.ErrorIs(t, err, mcDomain.ErrTemplateImageTypeUnsupported)

	huge := append(append([]byte{}, pngBytes...), make([]byte, 5<<20)...)
	_, err = uc.Execute(ctx, bytes.NewReader(huge))
	assert.ErrorIs(t, err, mcDomain.ErrTemplateImageTooLarge)
}

func TestUploadTemplateImage_storageErrorPropagates(t *testing.T) {
	boom := errors.New("disk full")
	uc := appmc.NewUploadTemplateImageUseCase(&fakeImageStorage{err: boom})
	_, err := uc.Execute(tenant.WithTenantID(context.Background(), "tenant-1"), bytes.NewReader(pngBytes))
	assert.ErrorIs(t, err, boom)
}
