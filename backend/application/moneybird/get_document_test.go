package moneybird_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appmb "github.com/adirkuhn/commerceos/application/moneybird"
	mb "github.com/adirkuhn/commerceos/domain/moneybird"
)

func TestGetDocument_statusOfAnOrder(t *testing.T) {
	doc := mb.Document{SourceType: mb.SourceSalesOrder, SourceID: "so-1", MoneybirdID: "inv-1", Status: mb.DocumentSynced}
	settings := configuredSettings()
	settings.AutoSync = true
	uc := appmb.NewGetDocumentUseCase(mockDB(t, 2), connected(t, "tok"), newFakeSettings(settings), newFakeDocuments(doc), testSecret)

	got, err := uc.Execute(tenantCtx(), mb.SourceSalesOrder, "so-1")

	require.NoError(t, err)
	require.NotNil(t, got.Document)
	assert.Equal(t, "inv-1", got.Document.MoneybirdID)
	assert.Equal(t, "adm-1", got.AdministrationID)
	assert.True(t, got.AutoSync)
}

func TestGetDocument_neverSent(t *testing.T) {
	uc := appmb.NewGetDocumentUseCase(mockDB(t, 2), connected(t, "tok"), newFakeSettings(configuredSettings()), newFakeDocuments(), testSecret)
	got, err := uc.Execute(tenantCtx(), mb.SourceQuote, "q-1")
	require.NoError(t, err)
	assert.Nil(t, got.Document)
}

func TestGetDocument_notConnected(t *testing.T) {
	uc := appmb.NewGetDocumentUseCase(mockDB(t, 2), &fakeConnections{}, newFakeSettings(), newFakeDocuments(), testSecret)
	_, err := uc.Execute(tenantCtx(), mb.SourceQuote, "q-1")
	assert.True(t, errors.Is(err, mb.ErrNotConnected))
}
