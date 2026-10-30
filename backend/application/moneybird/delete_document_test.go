package moneybird_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	appmb "github.com/adirkuhn/commerceos/application/moneybird"
	mb "github.com/adirkuhn/commerceos/domain/moneybird"
)

func newDeleteUC(t *testing.T, client *fakeClient, docs *fakeDocuments, outbox *eventrecordtest.Outbox) *appmb.DeleteDocumentUseCase {
	return appmb.NewDeleteDocumentUseCase(mockDB(t, 3), connected(t, "tok"), newFakeSettings(configuredSettings()), docs,
		&fakeFactory{client: client}, outbox, testSecret)
}

// An order booked with the wrong VAT: deleted in Moneybird and unlinked, so it can be sent again.
func TestDeleteDocument_deletesTheInvoiceAndTheLink(t *testing.T) {
	client, outbox := &fakeClient{}, &eventrecordtest.Outbox{}
	docs := newFakeDocuments(mb.Document{SourceType: mb.SourceSalesOrder, SourceID: "so-1", MoneybirdID: "inv-1", Reference: "CO-1002", Status: mb.DocumentSynced})

	require.NoError(t, newDeleteUC(t, client, docs, outbox).Execute(tenantCtx(), mb.SourceSalesOrder, "so-1"))

	assert.Equal(t, []string{"inv-1"}, client.deletedInvoices)
	assert.Empty(t, client.deletedEstimates)
	assert.Empty(t, docs.byKey)
	assert.Equal(t, []string{"integrations.moneybird_document.deleted"}, outbox.Subjects())
}

func TestDeleteDocument_quoteDeletesTheEstimate(t *testing.T) {
	client := &fakeClient{}
	docs := newFakeDocuments(mb.Document{SourceType: mb.SourceQuote, SourceID: "q-1", MoneybirdID: "est-1", Status: mb.DocumentSynced})

	require.NoError(t, newDeleteUC(t, client, docs, &eventrecordtest.Outbox{}).Execute(tenantCtx(), mb.SourceQuote, "q-1"))

	assert.Equal(t, []string{"est-1"}, client.deletedEstimates)
}

// A failed send that never created anything: only the link goes.
func TestDeleteDocument_failedSendWithoutDocumentOnlyClearsTheLink(t *testing.T) {
	client := &fakeClient{}
	docs := newFakeDocuments(mb.Document{SourceType: mb.SourceSalesOrder, SourceID: "so-1", Status: mb.DocumentFailed, Error: "boom"})

	require.NoError(t, newDeleteUC(t, client, docs, &eventrecordtest.Outbox{}).Execute(tenantCtx(), mb.SourceSalesOrder, "so-1"))

	assert.Empty(t, client.deletedInvoices)
	assert.Empty(t, docs.byKey)
}

// Moneybird refused (e.g. the invoice can't be deleted there): the link stays.
func TestDeleteDocument_moneybirdRefusesKeepsTheLink(t *testing.T) {
	client, outbox := &fakeClient{deleteErr: mb.ErrRequestFailed}, &eventrecordtest.Outbox{}
	docs := newFakeDocuments(mb.Document{SourceType: mb.SourceSalesOrder, SourceID: "so-1", MoneybirdID: "inv-1", Status: mb.DocumentSynced})

	err := newDeleteUC(t, client, docs, outbox).Execute(tenantCtx(), mb.SourceSalesOrder, "so-1")

	assert.True(t, errors.Is(err, mb.ErrRequestFailed))
	assert.Len(t, docs.byKey, 1)
	assert.Empty(t, outbox.Subjects())
}

func TestDeleteDocument_nothingSent(t *testing.T) {
	err := newDeleteUC(t, &fakeClient{}, newFakeDocuments(), &eventrecordtest.Outbox{}).Execute(tenantCtx(), mb.SourceSalesOrder, "so-1")
	assert.True(t, errors.Is(err, mb.ErrDocumentNotFound))
}
