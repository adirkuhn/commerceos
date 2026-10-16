package procurement_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	"github.com/adirkuhn/commerceos/domain/messaging"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	suppliersDomain "github.com/adirkuhn/commerceos/domain/suppliers"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// ── Stubs ────────────────────────────────────────────────────────────────────

type stubSupplierContacts struct {
	byID map[string]*suppliersDomain.Supplier
}

func (s *stubSupplierContacts) FindByID(_ context.Context, id string) (*suppliersDomain.Supplier, error) {
	if sup, ok := s.byID[id]; ok {
		return sup, nil
	}
	return nil, suppliersDomain.ErrSupplierNotFound
}

type stubSupplierIssueStateRepo struct {
	states    []procurementDomain.SupplierIssueState
	upserted  []procurementDomain.SupplierIssueState
	upsertErr error
}

func (s *stubSupplierIssueStateRepo) ListByPurchaseOrder(_ context.Context, _ string) ([]procurementDomain.SupplierIssueState, error) {
	return s.states, nil
}
func (s *stubSupplierIssueStateRepo) Upsert(_ context.Context, st *procurementDomain.SupplierIssueState) error {
	if s.upsertErr != nil {
		return s.upsertErr
	}
	s.upserted = append(s.upserted, *st)
	return nil
}

type stubMailer struct {
	sent []messaging.Message
	err  error
}

func (m *stubMailer) Send(_ context.Context, msg messaging.Message) error {
	if m.err != nil {
		return m.err
	}
	m.sent = append(m.sent, msg)
	return nil
}

// ── Fixture ──────────────────────────────────────────────────────────────────

// A PO for 10 × sku-1 and 5 × sku-2; the one completed delivery brought 7
// of sku-1 and all of sku-2, two of which were damaged.
type supplierIssueFixture struct {
	deps     appprocurement.SupplierIssueDeps
	states   *stubSupplierIssueStateRepo
	bookings *stubBookingRepo
	outbox   *eventrecordtest.Outbox
}

var (
	shortRef   = procurementDomain.SupplierIssueRef{Source: procurementDomain.SupplierIssueSourceBookingShortfall, SourceID: "line-1"}
	damagedRef = procurementDomain.SupplierIssueRef{Source: procurementDomain.SupplierIssueSourceBookingScan, SourceID: "scan-3"}
)

func newSupplierIssueFixture() *supplierIssueFixture {
	completedAt := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	sku1, sku2 := "sku-1", "sku-2"
	states := &stubSupplierIssueStateRepo{}
	bookings := &stubBookingRepo{completed: []procurementDomain.Booking{
		{ID: "booking-1", PurchaseOrderID: "po-1", Status: procurementDomain.BookingStatusCompleted, CompletedAt: &completedAt},
	}}
	f := &supplierIssueFixture{states: states, bookings: bookings, outbox: &eventrecordtest.Outbox{}}
	f.deps = appprocurement.SupplierIssueDeps{
		PurchaseOrders: &stubPurchaseOrderRepo{findResult: &procurementDomain.PurchaseOrder{ID: "po-1", SupplierID: strPtr("sup-1")}},
		Suppliers:      &stubSupplierContacts{byID: map[string]*suppliersDomain.Supplier{"sup-1": {ID: "sup-1", Name: "Bauducco NL", Email: "orders@bauducco.example"}}},
		LineItems: &stubLineItemRepo{byPOMap: map[string][]procurementDomain.PurchaseOrderLineItem{"po-1": {
			{ID: "line-1", PurchaseOrderID: "po-1", SKUID: &sku1, RawSKU: "B-100", RawDescription: "Pao de queijo 1kg", Quantity: 10, LineType: procurementDomain.LineItemTypeProduct},
			{ID: "line-2", PurchaseOrderID: "po-1", SKUID: &sku2, RawSKU: "B-200", RawDescription: "Guarana 2L", Quantity: 5, LineType: procurementDomain.LineItemTypeProduct},
		}}},
		Bookings: bookings,
		Scans: &stubBookingScanRepo{byBookingMap: map[string][]procurementDomain.BookingScan{"booking-1": {
			{ID: "scan-1", SKUID: sku1, ActionType: procurementDomain.BookingScanActionReceive, Quantity: 7},
			{ID: "scan-2", SKUID: sku2, ActionType: procurementDomain.BookingScanActionReceive, Quantity: 3},
			{ID: "scan-3", SKUID: sku2, ActionType: procurementDomain.BookingScanActionDamaged, WriteOffQuantity: 2, Reason: "crushed"},
		}}},
		ReceivingRecords: &stubReceivingRecordRepo{},
		Discrepancies:    &stubReceivingDiscrepancyRepo{},
		States:           states,
		SKUs:             &stubSKURepo{findByIDs: map[string]*pmsDomain.SKU{"sku-1": {ID: "sku-1", VariantID: "var-1", SKUCode: "PDQ-1KG"}}},
		Variants:         &stubVariantRepo{byID: map[string]*pmsDomain.Variant{"var-1": {ID: "var-1", ProductID: "prod-1"}}},
		Products:         &stubProductRepo{byID: map[string]*pmsDomain.Product{"prod-1": {ID: "prod-1", Name: "Pão de queijo"}}},
		Outbox:           f.outbox,
		Now:              func() time.Time { return time.Date(2026, 10, 16, 8, 0, 0, 0, time.UTC) },
	}
	return f
}

func asUser(userID string) context.Context {
	return tenant.WithUserID(context.Background(), userID)
}

// ── List ─────────────────────────────────────────────────────────────────────

func TestListSupplierIssues_returnsIssuesWithSupplierAndProductNames(t *testing.T) {
	f := newSupplierIssueFixture()
	uc := appprocurement.NewListSupplierIssuesUseCase(newMockGormDB(t), f.deps)

	got, err := uc.Execute(context.Background(), "po-1")

	require.NoError(t, err)
	require.Equal(t, "po-1", got.PurchaseOrderID)
	require.NotNil(t, got.Supplier)
	assert.Equal(t, appprocurement.SupplierContact{ID: "sup-1", Name: "Bauducco NL", Email: "orders@bauducco.example"}, *got.Supplier)
	assert.False(t, got.BookingInProgress)
	require.Len(t, got.Issues, 2)
	assert.Equal(t, shortRef, got.Issues[0].Ref)
	assert.Equal(t, int64(3), got.Issues[0].Quantity)
	assert.Equal(t, "PDQ-1KG", *got.Issues[0].SKUCode)
	assert.Equal(t, "Pão de queijo", *got.Issues[0].ProductName)
	assert.Equal(t, damagedRef, got.Issues[1].Ref)
	assert.Nil(t, got.Issues[1].SKUCode, "unknown SKU still listed, by the supplier's own text")
	assert.Equal(t, "Guarana 2L", got.Issues[1].SupplierDescription)
}

func TestListSupplierIssues_flagsADeliveryStillBeingBooked(t *testing.T) {
	f := newSupplierIssueFixture()
	f.bookings.activeResult = &procurementDomain.Booking{ID: "booking-2", Status: procurementDomain.BookingStatusInProgress}
	uc := appprocurement.NewListSupplierIssuesUseCase(newMockGormDB(t), f.deps)

	got, err := uc.Execute(context.Background(), "po-1")

	require.NoError(t, err)
	assert.True(t, got.BookingInProgress)
}

func TestListSupplierIssues_purchaseOrderWithoutSupplierHasNoContact(t *testing.T) {
	f := newSupplierIssueFixture()
	f.deps.PurchaseOrders = &stubPurchaseOrderRepo{findResult: &procurementDomain.PurchaseOrder{ID: "po-1"}}
	uc := appprocurement.NewListSupplierIssuesUseCase(newMockGormDB(t), f.deps)

	got, err := uc.Execute(context.Background(), "po-1")

	require.NoError(t, err)
	assert.Nil(t, got.Supplier)
}

func TestListSupplierIssues_unknownPurchaseOrder(t *testing.T) {
	f := newSupplierIssueFixture()
	f.deps.PurchaseOrders = &stubPurchaseOrderRepo{findErr: procurementDomain.ErrPurchaseOrderNotFound}
	uc := appprocurement.NewListSupplierIssuesUseCase(newMockGormDB(t), f.deps)

	_, err := uc.Execute(context.Background(), "po-x")

	require.ErrorIs(t, err, procurementDomain.ErrPurchaseOrderNotFound)
}

// ── Update status ────────────────────────────────────────────────────────────

func TestUpdateSupplierIssueStatus_storesStatusAndRecordsUpdated(t *testing.T) {
	f := newSupplierIssueFixture()
	uc := appprocurement.NewUpdateSupplierIssueStatusUseCase(newMockGormDB(t), f.deps)

	got, err := uc.Execute(asUser("user-1"), appprocurement.UpdateSupplierIssueStatusInput{
		PurchaseOrderID: "po-1", Ref: damagedRef, Status: procurementDomain.SupplierIssueStatusResolved, Note: "credit note CN-7",
	})

	require.NoError(t, err)
	assert.Equal(t, procurementDomain.SupplierIssueStatusResolved, got.Status)
	assert.Equal(t, "credit note CN-7", got.StatusNote)
	require.Len(t, f.states.upserted, 1)
	st := f.states.upserted[0]
	assert.Equal(t, "po-1", st.PurchaseOrderID)
	assert.Equal(t, damagedRef, st.Ref)
	assert.Equal(t, "user-1", st.UpdatedBy)
	assert.Equal(t, f.deps.Now(), st.UpdatedAt)
	e := onlyEvent(t, f.outbox)
	assert.Equal(t, "procurement.supplier_issue.updated", e.Subject)
	assert.Equal(t, "scan-3", e.Payload["entity_id"])
	assert.Equal(t, map[string]any{"purchase_order_id": "po-1", "source": "booking_scan", "kind": "damaged"}, e.Data())
	assert.Equal(t, map[string][2]any{"status": {"open", "resolved"}, "note": {"", "credit note CN-7"}}, e.Changes())
}

func TestUpdateSupplierIssueStatus_unchangedWritesNothing(t *testing.T) {
	f := newSupplierIssueFixture()
	f.states.states = []procurementDomain.SupplierIssueState{{Ref: damagedRef, Status: procurementDomain.SupplierIssueStatusClaimed, Note: "sent"}}
	uc := appprocurement.NewUpdateSupplierIssueStatusUseCase(newMockGormDB(t), f.deps)

	_, err := uc.Execute(context.Background(), appprocurement.UpdateSupplierIssueStatusInput{
		PurchaseOrderID: "po-1", Ref: damagedRef, Status: procurementDomain.SupplierIssueStatusClaimed, Note: "sent",
	})

	require.NoError(t, err)
	assert.Empty(t, f.states.upserted)
	assert.Empty(t, f.outbox.Subjects())
}

func TestUpdateSupplierIssueStatus_rejectsInvalidInput(t *testing.T) {
	for name, tc := range map[string]struct {
		in   appprocurement.UpdateSupplierIssueStatusInput
		want error
	}{
		"bad status": {appprocurement.UpdateSupplierIssueStatusInput{PurchaseOrderID: "po-1", Ref: damagedRef, Status: "lost"}, procurementDomain.ErrInvalidSupplierIssueStatus},
		"bad source": {appprocurement.UpdateSupplierIssueStatusInput{PurchaseOrderID: "po-1",
			Ref: procurementDomain.SupplierIssueRef{Source: "invoice", SourceID: "x"}, Status: procurementDomain.SupplierIssueStatusClaimed}, procurementDomain.ErrSupplierIssueNotFound},
		"not an issue of this PO": {appprocurement.UpdateSupplierIssueStatusInput{PurchaseOrderID: "po-1",
			Ref: procurementDomain.SupplierIssueRef{Source: procurementDomain.SupplierIssueSourceBookingScan, SourceID: "scan-1"}, Status: procurementDomain.SupplierIssueStatusClaimed}, procurementDomain.ErrSupplierIssueNotFound},
		"note too long": {appprocurement.UpdateSupplierIssueStatusInput{PurchaseOrderID: "po-1", Ref: damagedRef,
			Status: procurementDomain.SupplierIssueStatusClaimed, Note: string(make([]byte, 1001))}, procurementDomain.ErrSupplierIssueNoteTooLong},
	} {
		t.Run(name, func(t *testing.T) {
			f := newSupplierIssueFixture()
			uc := appprocurement.NewUpdateSupplierIssueStatusUseCase(newMockGormDB(t), f.deps)

			_, err := uc.Execute(context.Background(), tc.in)

			require.ErrorIs(t, err, tc.want)
			assert.Empty(t, f.states.upserted)
			assert.Empty(t, f.outbox.Subjects())
		})
	}
}

// ── Email ────────────────────────────────────────────────────────────────────

func emailInput(refs ...procurementDomain.SupplierIssueRef) appprocurement.EmailSupplierIssuesInput {
	return appprocurement.EmailSupplierIssuesInput{
		PurchaseOrderID: "po-1", To: "orders@bauducco.example", Subject: "Delivery problems",
		Body: "Hello,\n3 × Pão de queijo missing <check>", Refs: refs,
	}
}

func TestEmailSupplierIssues_sendsTheMessageAndClaimsOpenIssues(t *testing.T) {
	f := newSupplierIssueFixture()
	// Already resolved: emailing it again must not reopen it as claimed.
	f.states.states = []procurementDomain.SupplierIssueState{{Ref: damagedRef, Status: procurementDomain.SupplierIssueStatusResolved}}
	mailer := &stubMailer{}
	uc := appprocurement.NewEmailSupplierIssuesUseCase(newMockGormDBN(t, 2), f.deps, mailer)

	got, err := uc.Execute(asUser("user-1"), emailInput(shortRef, damagedRef))

	require.NoError(t, err)
	assert.Equal(t, appprocurement.EmailSupplierIssuesResult{SentTo: "orders@bauducco.example", Claimed: 1}, *got)
	require.Len(t, mailer.sent, 1)
	msg := mailer.sent[0]
	assert.Equal(t, messaging.ChannelEmail, msg.Channel)
	assert.Equal(t, "orders@bauducco.example", msg.To)
	assert.Equal(t, "Delivery problems", msg.Subject)
	assert.Contains(t, msg.HTMLBody, "3 × Pão de queijo missing &lt;check&gt;", "plain text is escaped, never sent as HTML")
	assert.Contains(t, msg.HTMLBody, "white-space:pre-wrap")
	require.Len(t, f.states.upserted, 1)
	assert.Equal(t, shortRef, f.states.upserted[0].Ref)
	assert.Equal(t, procurementDomain.SupplierIssueStatusClaimed, f.states.upserted[0].Status)
	assert.Equal(t, "user-1", f.states.upserted[0].UpdatedBy)
	assert.Equal(t, []string{"procurement.supplier_issue.updated", "procurement.supplier_issue_report.emailed"}, f.outbox.Subjects())
	emailed := f.outbox.Events[1]
	assert.Equal(t, "po-1", emailed.Payload["entity_id"])
	assert.Equal(t, map[string]any{"to": "orders@bauducco.example", "issue_count": float64(2), "claimed_count": float64(1)}, emailed.Data())
}

func TestEmailSupplierIssues_rejectsBadInputBeforeSending(t *testing.T) {
	for name, tc := range map[string]struct {
		in   appprocurement.EmailSupplierIssuesInput
		want error
	}{
		"bad address": {func() appprocurement.EmailSupplierIssuesInput {
			in := emailInput(shortRef)
			in.To = "not-an-email"
			return in
		}(), procurementDomain.ErrInvalidSupplierEmail},
		"empty subject": {func() appprocurement.EmailSupplierIssuesInput {
			in := emailInput(shortRef)
			in.Subject = "  "
			return in
		}(), procurementDomain.ErrSupplierEmailIncomplete},
		"empty body": {func() appprocurement.EmailSupplierIssuesInput {
			in := emailInput(shortRef)
			in.Body = ""
			return in
		}(), procurementDomain.ErrSupplierEmailIncomplete},
		"no issues":     {emailInput(), procurementDomain.ErrNoSupplierIssuesSelected},
		"unknown issue": {emailInput(procurementDomain.SupplierIssueRef{Source: procurementDomain.SupplierIssueSourceBookingScan, SourceID: "scan-1"}), procurementDomain.ErrSupplierIssueNotFound},
	} {
		t.Run(name, func(t *testing.T) {
			f := newSupplierIssueFixture()
			mailer := &stubMailer{}
			uc := appprocurement.NewEmailSupplierIssuesUseCase(newMockGormDBN(t, 2), f.deps, mailer)

			_, err := uc.Execute(context.Background(), tc.in)

			require.ErrorIs(t, err, tc.want)
			assert.Empty(t, mailer.sent)
			assert.Empty(t, f.states.upserted)
			assert.Empty(t, f.outbox.Subjects())
		})
	}
}

func TestEmailSupplierIssues_sendFailureClaimsNothing(t *testing.T) {
	f := newSupplierIssueFixture()
	mailer := &stubMailer{err: errors.New("smtp: 550 mailbox unavailable")}
	uc := appprocurement.NewEmailSupplierIssuesUseCase(newMockGormDBN(t, 2), f.deps, mailer)

	_, err := uc.Execute(context.Background(), emailInput(shortRef))

	require.ErrorIs(t, err, procurementDomain.ErrSupplierEmailNotSent)
	assert.Empty(t, f.states.upserted)
	assert.Empty(t, f.outbox.Subjects())
}
