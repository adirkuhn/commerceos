package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

type myParcelFulfilmentOrderModel struct {
	ID           string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID     string `gorm:"not null;type:uuid"`
	ConnectionID string `gorm:"column:connection_id;not null;type:uuid"`
	SalesOrderID string `gorm:"column:sales_order_id;not null;type:uuid"`
	MyParcelUUID string `gorm:"column:myparcel_uuid;not null"`
	Status       string `gorm:"not null"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (myParcelFulfilmentOrderModel) TableName() string { return "myparcel_fulfilment_orders" }

type MyParcelFulfilmentOrderRepo struct{ db *gorm.DB }

func NewMyParcelFulfilmentOrderRepo(db *gorm.DB) *MyParcelFulfilmentOrderRepo {
	return &MyParcelFulfilmentOrderRepo{db: db}
}

func (r *MyParcelFulfilmentOrderRepo) Create(ctx context.Context, o *shippingDomain.MyParcelFulfilmentOrder) error {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return errors.New("myParcelFulfilmentOrderRepo: operation called outside a transaction")
	}
	m := myParcelFulfilmentOrderModel{
		TenantID: o.TenantID, ConnectionID: o.ConnectionID, SalesOrderID: o.SalesOrderID,
		MyParcelUUID: o.MyParcelUUID, Status: o.Status, CreatedAt: o.CreatedAt, UpdatedAt: o.UpdatedAt,
	}
	if err := tx.WithContext(ctx).Create(&m).Error; err != nil {
		return fmt.Errorf("myParcelFulfilmentOrderRepo.Create: %w", err)
	}
	o.ID = m.ID
	return nil
}

func (r *MyParcelFulfilmentOrderRepo) FindByConnectionAndSalesOrder(ctx context.Context, connectionID, salesOrderID string) (*shippingDomain.MyParcelFulfilmentOrder, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("myParcelFulfilmentOrderRepo: operation called outside a transaction")
	}
	var m myParcelFulfilmentOrderModel
	err := tx.WithContext(ctx).Where("connection_id = ? AND sales_order_id = ?", connectionID, salesOrderID).First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("myParcelFulfilmentOrderRepo.FindByConnectionAndSalesOrder: %w", err)
	}
	return &shippingDomain.MyParcelFulfilmentOrder{
		ID: m.ID, TenantID: m.TenantID, ConnectionID: m.ConnectionID, SalesOrderID: m.SalesOrderID,
		MyParcelUUID: m.MyParcelUUID, Status: m.Status, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}, nil
}

func (r *MyParcelFulfilmentOrderRepo) FindByMyParcelUUID(ctx context.Context, myParcelUUID string) (*shippingDomain.MyParcelFulfilmentOrder, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("myParcelFulfilmentOrderRepo: operation called outside a transaction")
	}
	var m myParcelFulfilmentOrderModel
	err := tx.WithContext(ctx).Where("myparcel_uuid = ?", myParcelUUID).First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("myParcelFulfilmentOrderRepo.FindByMyParcelUUID: %w", err)
	}
	return &shippingDomain.MyParcelFulfilmentOrder{
		ID: m.ID, TenantID: m.TenantID, ConnectionID: m.ConnectionID, SalesOrderID: m.SalesOrderID,
		MyParcelUUID: m.MyParcelUUID, Status: m.Status, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}, nil
}
