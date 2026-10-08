package channels

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// ListBolShipmentConfirmationsUseCase returns every attempt to confirm a
// sales order's shipment on Bol, newest first -- what the order page shows
// so the merchant can see whether Bol accepted it.
type ListBolShipmentConfirmationsUseCase struct {
	db             *gorm.DB
	salesOrderRepo omsDomain.SalesOrderRepository
	confirmations  channelsDomain.BolShipmentConfirmationRepository
}

func NewListBolShipmentConfirmationsUseCase(db *gorm.DB, salesOrderRepo omsDomain.SalesOrderRepository, confirmations channelsDomain.BolShipmentConfirmationRepository) *ListBolShipmentConfirmationsUseCase {
	return &ListBolShipmentConfirmationsUseCase{db: db, salesOrderRepo: salesOrderRepo, confirmations: confirmations}
}

func (uc *ListBolShipmentConfirmationsUseCase) Execute(ctx context.Context, salesOrderID string) ([]channelsDomain.BolShipmentConfirmation, error) {
	out := []channelsDomain.BolShipmentConfirmation{}
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if _, err := uc.salesOrderRepo.FindByID(ctx, salesOrderID); err != nil {
			return err
		}
		found, err := uc.confirmations.ListBySalesOrder(ctx, salesOrderID)
		if err != nil {
			return err
		}
		out = append(out, found...)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("listBolShipmentConfirmations: %w", err)
	}
	return out, nil
}
