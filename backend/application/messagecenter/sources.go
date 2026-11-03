package messagecenter

import (
	"context"
	"errors"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"

	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
)

// salesOrderFinder is oms.SalesOrderRepository narrowed to point of use.
type salesOrderFinder interface {
	FindByID(ctx context.Context, id string) (*omsDomain.SalesOrder, error)
}

// customerFinder is customers.CustomerRepository narrowed to point of use.
type customerFinder interface {
	FindByID(ctx context.Context, id string) (*customersDomain.Customer, error)
}

type lineItemLister interface {
	ListBySalesOrder(ctx context.Context, salesOrderID string) ([]omsDomain.SalesOrderLineItem, error)
}

type shipmentLister interface {
	FindBySalesOrderID(ctx context.Context, salesOrderID string) ([]shippingDomain.Shipment, error)
}

type carrierOverrideFinder interface {
	FindByConnectionAndCarrier(ctx context.Context, connectionID string, carrier shippingDomain.Carrier) (*shippingDomain.ShippingCarrierOverride, error)
}

// originFinder is application/integrations.SalesOrderOrigins narrowed to
// reading an order's origin.
type originFinder interface {
	Of(ctx context.Context, salesOrderID string) (*integrationsDomain.SalesOrderOrigin, error)
}

type companyProfileReader interface {
	Get(ctx context.Context) (*settingsDomain.CompanyProfile, error)
}

type tenantSettingsReader interface {
	Get(ctx context.Context) (*settingsDomain.TenantSettings, error)
}

// OrderSources load what a message about a sales order shows (see
// loadedOrder). Reads only, inside the engine's tenant transaction.
type OrderSources struct {
	SalesOrders      salesOrderFinder
	Origins          originFinder
	LineItems        lineItemLister
	Customers        customerFinder
	Shipments        shipmentLister
	CarrierOverrides carrierOverrideFinder
}

// StoreSources load the store's own details and formatting settings.
type StoreSources struct {
	CompanyProfile companyProfileReader
	Settings       tenantSettingsReader
}

// load reads one order; nil (no error) when it no longer exists.
func (s OrderSources) load(ctx context.Context, orderID string) (*loadedOrder, error) {
	order, err := s.SalesOrders.FindByID(ctx, orderID)
	if errors.Is(err, omsDomain.ErrSalesOrderNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	origin, err := s.Origins.Of(ctx, orderID)
	if err != nil {
		return nil, err
	}
	lo := &loadedOrder{order: order, source: string(origin.Origin)}
	if order.CustomerID != nil {
		if lo.customer, err = s.Customers.FindByID(ctx, *order.CustomerID); err != nil {
			return nil, err
		}
	}
	if lo.lines, err = s.LineItems.ListBySalesOrder(ctx, orderID); err != nil {
		return nil, err
	}
	shipments, err := s.Shipments.FindBySalesOrderID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	// newest first: the latest shipment still active is the one to track
	for i := range shipments {
		if shipments[i].IsActive() {
			lo.shipment = &shipments[i]
			break
		}
	}
	if lo.shipment != nil {
		if lo.trackingURL, err = s.trackingURL(ctx, *lo.shipment, order.Shipping.Postcode); err != nil {
			return nil, err
		}
	}
	return lo, nil
}

// trackingURL prefers the connection's own carrier tracking link template
// over MyParcel's link.
func (s OrderSources) trackingURL(ctx context.Context, sh shippingDomain.Shipment, postcode string) (string, error) {
	override, err := s.CarrierOverrides.FindByConnectionAndCarrier(ctx, sh.ConnectionID, sh.Carrier)
	if err != nil {
		return "", err
	}
	if override != nil {
		if url := shippingDomain.BuildCarrierTrackingURL(override.TrackingURLTemplate, sh.Barcode, postcode); url != "" {
			return url, nil
		}
	}
	return sh.TrackingURL, nil
}
