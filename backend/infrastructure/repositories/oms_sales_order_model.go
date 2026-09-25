package repositories

import (
	"time"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
)

// ── GORM models ──────────────────────────────────────────────────────────────

type salesOrderModel struct {
	ID              string  `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID        string  `gorm:"not null;type:uuid"`
	Source          string  `gorm:"not null"`
	ExternalOrderID string  `gorm:"column:external_order_id;not null"`
	ConnectionID    *string `gorm:"column:connection_id;type:uuid"`
	SalesChannelID  *string `gorm:"column:sales_channel_id;type:uuid"`
	Number          string
	OrderKey        string `gorm:"column:order_key"`
	Status          string `gorm:"not null;default:completed"`

	WarehouseID string  `gorm:"not null;type:uuid"`
	CustomerID  *string `gorm:"type:uuid"`

	BillingFirstName  string
	BillingLastName   string
	BillingCompany    string
	BillingAddress1   string `gorm:"column:billing_address_1"`
	BillingAddress2   string `gorm:"column:billing_address_2"`
	BillingCity       string
	BillingState      string
	BillingPostcode   string
	BillingCountry    string
	BillingEmail      string
	BillingPhone      string
	ShippingFirstName string
	ShippingLastName  string
	ShippingCompany   string
	ShippingAddress1  string `gorm:"column:shipping_address_1"`
	ShippingAddress2  string `gorm:"column:shipping_address_2"`
	ShippingCity      string
	ShippingState     string
	ShippingPostcode  string
	ShippingCountry   string
	ShippingPhone     string
	CustomerNote      string

	CurrencyCode            string `gorm:"not null;default:'USD'"`
	DiscountTotalMinorUnits int64  `gorm:"not null;default:0"`
	ShippingTotalMinorUnits int64  `gorm:"not null;default:0"`
	TaxTotalMinorUnits      int64  `gorm:"not null;default:0"`
	TotalMinorUnits         int64  `gorm:"not null;default:0"`

	PaymentMethod       string
	PaymentMethodTitle  string
	ShippingMethod      string `gorm:"column:shipping_method;not null;default:''"`
	ShippingMethodTitle string `gorm:"column:shipping_method_title;not null;default:''"`
	CouponCode          string `gorm:"column:coupon_code;not null;default:''"`
	TransactionID       string
	PaymentProvider     string `gorm:"column:payment_provider;not null;default:''"`
	PaymentExternalID   string `gorm:"column:payment_external_id;not null;default:''"`
	DatePaid            *time.Time
	DateCompleted       *time.Time

	OccurredAt time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time

	StockDecremented        bool `gorm:"not null;default:false"`
	NeedsStockReview        bool `gorm:"column:needs_stock_review;not null;default:false"`
	NeedsSKUReview          bool `gorm:"column:needs_sku_review;not null;default:false"`
	NeedsManualRefundReview bool `gorm:"column:needs_manual_refund_review;not null;default:false"`
	Fulfilled               bool `gorm:"not null;default:false"`
	Packed                  bool `gorm:"not null;default:false"`
}

func (salesOrderModel) TableName() string { return "sales_orders" }

func fromAddressDomain(a omsDomain.Address) (firstName, lastName, company, address1, address2, city, state, postcode, country, email, phone string) {
	return a.FirstName, a.LastName, a.Company, a.Address1, a.Address2, a.City, a.State, a.Postcode, a.Country, a.Email, a.Phone
}

func salesOrderModelFromDomain(o *omsDomain.SalesOrder) salesOrderModel {
	bFirst, bLast, bCompany, bAddr1, bAddr2, bCity, bState, bPostcode, bCountry, bEmail, bPhone := fromAddressDomain(o.Billing)
	sFirst, sLast, sCompany, sAddr1, sAddr2, sCity, sState, sPostcode, sCountry, _, sPhone := fromAddressDomain(o.Shipping)
	return salesOrderModel{
		ID: o.ID, TenantID: o.TenantID, Source: string(o.Source), ExternalOrderID: o.ExternalOrderID,
		ConnectionID: o.ConnectionID, SalesChannelID: o.SalesChannelID, Number: o.Number, OrderKey: o.OrderKey, Status: string(o.Status),
		WarehouseID: o.WarehouseID, CustomerID: o.CustomerID,
		BillingFirstName: bFirst, BillingLastName: bLast, BillingCompany: bCompany,
		BillingAddress1: bAddr1, BillingAddress2: bAddr2, BillingCity: bCity, BillingState: bState,
		BillingPostcode: bPostcode, BillingCountry: bCountry, BillingEmail: bEmail, BillingPhone: bPhone,
		ShippingFirstName: sFirst, ShippingLastName: sLast, ShippingCompany: sCompany,
		ShippingAddress1: sAddr1, ShippingAddress2: sAddr2, ShippingCity: sCity, ShippingState: sState,
		ShippingPostcode: sPostcode, ShippingCountry: sCountry, ShippingPhone: sPhone,
		CustomerNote: o.CustomerNote, CurrencyCode: o.CurrencyCode,
		DiscountTotalMinorUnits: o.DiscountTotalMinorUnits, ShippingTotalMinorUnits: o.ShippingTotalMinorUnits,
		TaxTotalMinorUnits: o.TaxTotalMinorUnits, TotalMinorUnits: o.TotalMinorUnits,
		PaymentMethod: o.PaymentMethod, PaymentMethodTitle: o.PaymentMethodTitle,
		ShippingMethod: o.ShippingMethod, ShippingMethodTitle: o.ShippingMethodTitle, CouponCode: o.CouponCode, TransactionID: o.TransactionID,
		PaymentProvider: o.PaymentProvider, PaymentExternalID: o.PaymentExternalID,
		DatePaid: o.DatePaid, DateCompleted: o.DateCompleted,
		OccurredAt: o.OccurredAt, CreatedAt: o.CreatedAt, UpdatedAt: o.UpdatedAt,
		StockDecremented: o.StockDecremented,
		NeedsStockReview: o.NeedsStockReview, NeedsSKUReview: o.NeedsSKUReview,
		NeedsManualRefundReview: o.NeedsManualRefundReview,
		Fulfilled:               o.Fulfilled,
		Packed:                  o.Packed,
	}
}

func toSalesOrderDomain(m salesOrderModel) *omsDomain.SalesOrder {
	return &omsDomain.SalesOrder{
		ID: m.ID, TenantID: m.TenantID, Source: omsDomain.SalesOrderSource(m.Source),
		ExternalOrderID: m.ExternalOrderID, ConnectionID: m.ConnectionID, SalesChannelID: m.SalesChannelID, Number: m.Number, OrderKey: m.OrderKey,
		Status: omsDomain.SalesOrderStatus(m.Status), WarehouseID: m.WarehouseID, CustomerID: m.CustomerID,
		Billing: omsDomain.Address{
			FirstName: m.BillingFirstName, LastName: m.BillingLastName, Company: m.BillingCompany,
			Address1: m.BillingAddress1, Address2: m.BillingAddress2, City: m.BillingCity, State: m.BillingState,
			Postcode: m.BillingPostcode, Country: m.BillingCountry, Email: m.BillingEmail, Phone: m.BillingPhone,
		},
		Shipping: omsDomain.Address{
			FirstName: m.ShippingFirstName, LastName: m.ShippingLastName, Company: m.ShippingCompany,
			Address1: m.ShippingAddress1, Address2: m.ShippingAddress2, City: m.ShippingCity, State: m.ShippingState,
			Postcode: m.ShippingPostcode, Country: m.ShippingCountry, Phone: m.ShippingPhone,
		},
		CustomerNote: m.CustomerNote, CurrencyCode: m.CurrencyCode,
		DiscountTotalMinorUnits: m.DiscountTotalMinorUnits, ShippingTotalMinorUnits: m.ShippingTotalMinorUnits,
		TaxTotalMinorUnits: m.TaxTotalMinorUnits, TotalMinorUnits: m.TotalMinorUnits,
		PaymentMethod: m.PaymentMethod, PaymentMethodTitle: m.PaymentMethodTitle,
		ShippingMethod: m.ShippingMethod, ShippingMethodTitle: m.ShippingMethodTitle, CouponCode: m.CouponCode, TransactionID: m.TransactionID,
		PaymentProvider: m.PaymentProvider, PaymentExternalID: m.PaymentExternalID,
		DatePaid: m.DatePaid, DateCompleted: m.DateCompleted,
		OccurredAt: m.OccurredAt, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
		StockDecremented: m.StockDecremented,
		NeedsStockReview: m.NeedsStockReview, NeedsSKUReview: m.NeedsSKUReview,
		NeedsManualRefundReview: m.NeedsManualRefundReview,
		Fulfilled:               m.Fulfilled,
		Packed:                  m.Packed,
	}
}
