package procurement_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubCreatePOForDocumentUC struct {
	out      *procurementDomain.PurchaseOrder
	err      error
	gotInput appprocurement.CreatePurchaseOrderInput
}

func (s *stubCreatePOForDocumentUC) Execute(_ context.Context, in appprocurement.CreatePurchaseOrderInput) (*procurementDomain.PurchaseOrder, error) {
	s.gotInput = in
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &procurementDomain.PurchaseOrder{ID: "po-1", SupplierID: in.SupplierID}, nil
}

type stubAttachDocumentUC struct {
	out      *procurementDomain.DocumentParsingJob
	err      error
	gotInput appprocurement.AttachDocumentToPurchaseOrderInput
}

func (s *stubAttachDocumentUC) Execute(_ context.Context, in appprocurement.AttachDocumentToPurchaseOrderInput) (*procurementDomain.DocumentParsingJob, error) {
	s.gotInput = in
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &procurementDomain.DocumentParsingJob{ID: "job-1", PurchaseOrderID: in.PurchaseOrderID}, nil
}

func TestCreatePurchaseOrderFromDocument_createsPOThenAttachesDocument(t *testing.T) {
	createPO := &stubCreatePOForDocumentUC{}
	attach := &stubAttachDocumentUC{}
	uc := appprocurement.NewCreatePurchaseOrderFromDocumentUseCase(createPO, attach)

	job, err := uc.Execute(context.Background(), appprocurement.CreatePurchaseOrderFromDocumentInput{
		SupplierID: strPtr("supplier-1"), WarehouseID: strPtr("warehouse-1"), CurrencyCode: "USD",
		FileName: "order.csv", ContentType: "text/csv", Data: strings.NewReader("data"),
	})

	require.NoError(t, err)
	require.NotNil(t, createPO.gotInput.SupplierID)
	assert.Equal(t, "supplier-1", *createPO.gotInput.SupplierID)
	assert.Equal(t, appprocurement.PurchaseOrderSourceDocument, createPO.gotInput.Source)
	assert.Equal(t, "po-1", attach.gotInput.PurchaseOrderID)
	assert.Equal(t, "order.csv", attach.gotInput.FileName)
	assert.Equal(t, "job-1", job.ID)
}

func TestCreatePurchaseOrderFromDocument_allowsOmittingSupplier(t *testing.T) {
	createPO := &stubCreatePOForDocumentUC{}
	attach := &stubAttachDocumentUC{}
	uc := appprocurement.NewCreatePurchaseOrderFromDocumentUseCase(createPO, attach)

	_, err := uc.Execute(context.Background(), appprocurement.CreatePurchaseOrderFromDocumentInput{
		WarehouseID: strPtr("warehouse-1"), CurrencyCode: "USD", FileName: "order.csv", Data: strings.NewReader("data"),
	})

	require.NoError(t, err)
	assert.Nil(t, createPO.gotInput.SupplierID)
}

func TestCreatePurchaseOrderFromDocument_propagatesCreatePOError(t *testing.T) {
	createPO := &stubCreatePOForDocumentUC{err: errors.New("invalid warehouse")}
	uc := appprocurement.NewCreatePurchaseOrderFromDocumentUseCase(createPO, &stubAttachDocumentUC{})

	_, err := uc.Execute(context.Background(), appprocurement.CreatePurchaseOrderFromDocumentInput{
		FileName: "order.csv", Data: strings.NewReader(""),
	})

	require.Error(t, err)
}

func TestCreatePurchaseOrderFromDocument_propagatesAttachError(t *testing.T) {
	attach := &stubAttachDocumentUC{err: errors.New("storage down")}
	uc := appprocurement.NewCreatePurchaseOrderFromDocumentUseCase(&stubCreatePOForDocumentUC{}, attach)

	_, err := uc.Execute(context.Background(), appprocurement.CreatePurchaseOrderFromDocumentInput{
		SupplierID: strPtr("supplier-1"), FileName: "order.csv", Data: strings.NewReader(""),
	})

	require.Error(t, err)
}
