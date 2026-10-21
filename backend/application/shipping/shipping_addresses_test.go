package shipping_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	appshipping "github.com/adirkuhn/commerceos/application/shipping"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func warehouseAddress() shippingDomain.ShippingAddress {
	return shippingDomain.ShippingAddress{Company: "Warehouse", Street: "Kade", Number: "9", PostalCode: "3011 aa", City: "Rotterdam", CC: "nl"}
}

func TestGetShippingAddresses_returnsSavedOnesAndTheCompanyFallback(t *testing.T) {
	sender := warehouseAddress()
	prefs := &fakeShippingPreferencesRepo{prefs: &shippingDomain.ShippingPreferences{SenderAddress: &sender}}
	uc := appshipping.NewGetShippingAddressesUseCase(newMockGormDB(t, 1), prefs, companyWithAddress())

	got, err := uc.Execute(adminCtx())

	require.NoError(t, err)
	require.NotNil(t, got.Sender)
	assert.Equal(t, "Warehouse", got.Sender.Company)
	assert.Nil(t, got.Return)
	require.NotNil(t, got.CompanyAddress)
	assert.Equal(t, shippingDomain.ShippingAddress{
		Company: "Cantinho Brasileiro", Street: "Damrak", Number: "1A", PostalCode: "1012AB", City: "Amsterdam", CC: "NL",
		Email: "shop@example.com", Phone: "+31612345678",
	}, *got.CompanyAddress)
}

func TestUpdateShippingAddresses_validatesSavesBothAndRecordsEvent(t *testing.T) {
	defaultID := "mp"
	prefs := &fakeShippingPreferencesRepo{prefs: &shippingDomain.ShippingPreferences{DefaultConnectionID: &defaultID}}
	outbox := &eventrecordtest.Outbox{}
	uc := appshipping.NewUpdateShippingAddressesUseCase(newMockGormDB(t, 1), prefs, outbox)
	sender, ret := warehouseAddress(), warehouseAddress()
	ret.Company = "Returns"

	err := uc.Execute(adminCtx(), appshipping.UpdateShippingAddressesInput{Sender: &sender, Return: &ret})

	require.NoError(t, err)
	require.NotNil(t, prefs.upserted)
	assert.Equal(t, "tenant-1", prefs.upserted.TenantID)
	assert.Equal(t, "3011AA", prefs.upserted.SenderAddress.PostalCode, "normalized")
	assert.Equal(t, "Returns", prefs.upserted.ReturnAddress.Company)
	assert.Same(t, &defaultID, prefs.upserted.DefaultConnectionID, "the default connection pick is kept")
	e, err := outbox.Only()
	require.NoError(t, err)
	assert.Equal(t, "shipping.addresses.updated", e.Subject)
	assert.Equal(t, "tenant-1", e.Payload["entity_id"])
}

func TestUpdateShippingAddresses_nilClearsAnAddress(t *testing.T) {
	sender := warehouseAddress()
	prefs := &fakeShippingPreferencesRepo{prefs: &shippingDomain.ShippingPreferences{SenderAddress: &sender, ReturnAddress: &sender}}
	uc := appshipping.NewUpdateShippingAddressesUseCase(newMockGormDB(t, 1), prefs, &eventrecordtest.Outbox{})

	require.NoError(t, uc.Execute(adminCtx(), appshipping.UpdateShippingAddressesInput{Sender: &sender}))

	assert.NotNil(t, prefs.upserted.SenderAddress)
	assert.Nil(t, prefs.upserted.ReturnAddress)
}

func TestUpdateShippingAddresses_rejectsIncompleteAddress(t *testing.T) {
	prefs := &fakeShippingPreferencesRepo{}
	uc := appshipping.NewUpdateShippingAddressesUseCase(nil, prefs, &eventrecordtest.Outbox{})
	bad := warehouseAddress()
	bad.Street = ""

	err := uc.Execute(adminCtx(), appshipping.UpdateShippingAddressesInput{Return: &bad})

	assert.ErrorIs(t, err, shippingDomain.ErrShippingAddressInvalid)
	assert.Nil(t, prefs.upserted)
}

func TestUpdateShippingAddresses_regularUserForbidden(t *testing.T) {
	prefs := &fakeShippingPreferencesRepo{}
	uc := appshipping.NewUpdateShippingAddressesUseCase(nil, prefs, &eventrecordtest.Outbox{})
	ctx := tenant.WithRole(tenant.WithTenantID(context.Background(), "tenant-1"), tenant.RoleCustomerUser)

	err := uc.Execute(ctx, appshipping.UpdateShippingAddressesInput{})

	assert.ErrorIs(t, err, shippingDomain.ErrShippingPreferencesForbidden)
}
