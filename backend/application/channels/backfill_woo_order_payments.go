package channels

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	appoms "github.com/adirkuhn/commerceos/application/oms"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// BackfillWooOrderPaymentsBatchSize is how many orders one call checks --
// each is a Woo API round trip, so a call stays well inside a request's
// time budget; the caller repeats with NextCursor until it comes back
// empty.
const BackfillWooOrderPaymentsBatchSize = 25

var ErrInvalidBackfillCursor = errors.New("invalid backfill cursor")

type BackfillWooOrderPaymentsInput struct {
	ConnectionID string
	// Cursor is the previous call's NextCursor -- empty starts from the
	// newest order.
	Cursor string
}

type BackfillWooOrderPaymentsResult struct {
	Checked int
	Updated int
	// Errors names each order that couldn't be fetched or updated -- one
	// failure never stops the rest of the batch.
	Errors []string
	// NextCursor is empty once there is nothing left to check.
	NextCursor string
}

// wooClientResolver gives the Woo API client for one connection (and
// rejects a connection that isn't WooCommerce).
type wooClientResolver interface {
	ClientFor(ctx context.Context, connectionID string) (integrationsDomain.WooClient, error)
}

// BackfillWooOrderPaymentsUseCase fills in payment info (method, title,
// transaction id, paid date) on Woo orders imported before the order
// import re-synced it -- e.g. imported while pending and paid later. It
// only touches orders that went through a payment yet carry neither a
// transaction id nor a paid date (see omsDomain.MissingPaymentInfoLister),
// fetches each from Woo, and writes payment fields only (see
// applyWooPaymentChanges) -- never status, addresses or stock.
type BackfillWooOrderPaymentsUseCase struct {
	db          *gorm.DB
	orders      omsDomain.MissingPaymentInfoLister
	clients     wooClientResolver
	updateOrder updateSalesOrderUC
}

func NewBackfillWooOrderPaymentsUseCase(db *gorm.DB, orders omsDomain.MissingPaymentInfoLister, clients wooClientResolver, updateOrder updateSalesOrderUC) *BackfillWooOrderPaymentsUseCase {
	return &BackfillWooOrderPaymentsUseCase{db: db, orders: orders, clients: clients, updateOrder: updateOrder}
}

func (uc *BackfillWooOrderPaymentsUseCase) Execute(ctx context.Context, in BackfillWooOrderPaymentsInput) (*BackfillWooOrderPaymentsResult, error) {
	after, err := decodeBackfillCursor(in.Cursor)
	if err != nil {
		return nil, fmt.Errorf("backfillWooOrderPayments: %w", err)
	}
	client, err := uc.clients.ClientFor(ctx, in.ConnectionID)
	if err != nil {
		return nil, fmt.Errorf("backfillWooOrderPayments: %w", err)
	}

	var batch []omsDomain.SalesOrder
	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		batch, err = uc.orders.ListMissingPaymentInfo(ctx, in.ConnectionID, after, BackfillWooOrderPaymentsBatchSize)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("backfillWooOrderPayments: %w", err)
	}

	result := &BackfillWooOrderPaymentsResult{}
	for _, order := range batch {
		result.Checked++
		updated, err := uc.backfillOne(ctx, client, order)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("woo order %s: %v", order.ExternalOrderID, err))
			continue
		}
		if updated {
			result.Updated++
		}
	}
	if len(batch) == BackfillWooOrderPaymentsBatchSize {
		last := batch[len(batch)-1]
		result.NextCursor = encodeBackfillCursor(omsDomain.MissingPaymentInfoCursor{OccurredAt: last.OccurredAt, ID: last.ID})
	}
	return result, nil
}

func (uc *BackfillWooOrderPaymentsUseCase) backfillOne(ctx context.Context, client integrationsDomain.WooClient, order omsDomain.SalesOrder) (bool, error) {
	wo, err := client.GetOrder(ctx, order.ExternalOrderID)
	if err != nil {
		return false, fmt.Errorf("fetch: %w", err)
	}
	in := appoms.UpdateSalesOrderInput{}
	if !applyWooPaymentChanges(order, *wo, &in) {
		return false, nil
	}
	if _, err := uc.updateOrder.Execute(ctx, order.ID, in); err != nil {
		return false, fmt.Errorf("update: %w", err)
	}
	return true, nil
}

// The cursor is opaque to callers: "<occurred_at unix nanos>_<order id>".
func encodeBackfillCursor(c omsDomain.MissingPaymentInfoCursor) string {
	return strconv.FormatInt(c.OccurredAt.UnixNano(), 10) + "_" + c.ID
}

func decodeBackfillCursor(s string) (*omsDomain.MissingPaymentInfoCursor, error) {
	if s == "" {
		return nil, nil
	}
	nanos, id, ok := strings.Cut(s, "_")
	if !ok || id == "" {
		return nil, ErrInvalidBackfillCursor
	}
	n, err := strconv.ParseInt(nanos, 10, 64)
	if err != nil {
		return nil, ErrInvalidBackfillCursor
	}
	return &omsDomain.MissingPaymentInfoCursor{OccurredAt: time.Unix(0, n).UTC(), ID: id}, nil
}
