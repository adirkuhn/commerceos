package integrations

import (
	"context"
	"errors"
	"fmt"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
)

// SalesOrderOrigins is how other modules (oms when it creates an order,
// channels when it imports or sends one) record and look up where a sales
// order came from (COM-15) without reaching into this domain's repository.
// It never opens a transaction: every call runs inside the caller's
// tenant.WithTenant, so the origin is written atomically with the order and
// under the event the caller records for it.
type SalesOrderOrigins struct {
	repo integrationsDomain.SalesOrderOriginRepository
}

func NewSalesOrderOrigins(repo integrationsDomain.SalesOrderOriginRepository) *SalesOrderOrigins {
	return &SalesOrderOrigins{repo: repo}
}

// Record validates in against its origin and saves it as the order's one
// origin row.
func (s *SalesOrderOrigins) Record(ctx context.Context, in integrationsDomain.SalesOrderOriginInput) error {
	o, err := integrationsDomain.NewSalesOrderOrigin(in)
	if err != nil {
		return fmt.Errorf("salesOrderOrigins.Record: %w", err)
	}
	if err := s.repo.Save(ctx, &o); err != nil {
		return fmt.Errorf("salesOrderOrigins.Record: %w", err)
	}
	return nil
}

// Of returns the order's origin, or ErrSalesOrderOriginNotFound.
func (s *SalesOrderOrigins) Of(ctx context.Context, salesOrderID string) (*integrationsDomain.SalesOrderOrigin, error) {
	return s.repo.FindBySalesOrderID(ctx, salesOrderID)
}

// SetExternalKey changes the order key kept on the order's origin.
func (s *SalesOrderOrigins) SetExternalKey(ctx context.Context, salesOrderID, key string) error {
	o, err := s.repo.FindBySalesOrderID(ctx, salesOrderID)
	if err != nil {
		return fmt.Errorf("salesOrderOrigins.SetExternalKey: %w", err)
	}
	o.ExternalKey = key
	if err := s.repo.Save(ctx, o); err != nil {
		return fmt.Errorf("salesOrderOrigins.SetExternalKey: %w", err)
	}
	return nil
}

// FindByExternalID finds an order from a system without a connection row
// (POS) by that system's own id, or ErrSalesOrderOriginNotFound.
func (s *SalesOrderOrigins) FindByExternalID(ctx context.Context, origin integrationsDomain.Origin, externalOrderID string) (*integrationsDomain.SalesOrderOrigin, error) {
	return s.repo.FindByOriginExternalID(ctx, origin, externalOrderID)
}

// FindImported finds a shop order by the connection it is being imported
// through and the shop's own order id. An order of that origin with no
// connection (migration 246 could not tell which connection it came from)
// is claimed by this one instead of being imported twice. Not known at all
// is ErrSalesOrderOriginNotFound.
func (s *SalesOrderOrigins) FindImported(ctx context.Context, origin integrationsDomain.Origin, connectionID, externalOrderID string) (*integrationsDomain.SalesOrderOrigin, error) {
	o, err := s.repo.FindByConnectionExternalID(ctx, connectionID, externalOrderID)
	if !errors.Is(err, integrationsDomain.ErrSalesOrderOriginNotFound) {
		return o, err
	}
	o, err = s.repo.FindByOriginExternalID(ctx, origin, externalOrderID)
	if err != nil {
		return nil, err
	}
	if err := o.ClaimConnection(connectionID); err != nil {
		return nil, fmt.Errorf("salesOrderOrigins.FindImported: %w", err)
	}
	if err := s.repo.Save(ctx, o); err != nil {
		return nil, fmt.Errorf("salesOrderOrigins.FindImported: %w", err)
	}
	return o, nil
}
