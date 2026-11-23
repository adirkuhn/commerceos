package assistanttools

import (
	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/application/assistant/tools"
	appchannels "github.com/adirkuhn/commerceos/application/channels"
	appcustomers "github.com/adirkuhn/commerceos/application/customers"
	appinventory "github.com/adirkuhn/commerceos/application/inventory"
	appoms "github.com/adirkuhn/commerceos/application/oms"
	apppms "github.com/adirkuhn/commerceos/application/pms"
	appsearch "github.com/adirkuhn/commerceos/application/search"
	appshipping "github.com/adirkuhn/commerceos/application/shipping"
	"github.com/adirkuhn/commerceos/domain/assistant"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
)

// salesTools: shipments, customers and per-warehouse stock.
func salesTools(db *gorm.DB) []assistant.Tool {
	salesOrderRepo := repositories.NewSalesOrderRepo(db)
	customerRepo := repositories.NewCustomerRepo(db)
	productRepo := repositories.NewProductRepo(db)
	movementRepo := repositories.NewStockMovementRepo(db)
	search := appsearch.NewGlobalSearchUseCase(db, productRepo, repositories.NewSupplierRepo(db), customerRepo, salesOrderRepo)
	orders := appoms.NewListSalesOrdersUseCase(db, salesOrderRepo, repositories.NewTenantSettingsRepo(db))

	return []assistant.Tool{
		tools.NewGetOrderShipmentsTool(
			appshipping.NewListShipmentsForOrderUseCase(db, repositories.NewShipmentRepo(db), repositories.NewShippingCarrierOverrideRepo(db)),
			appshipping.NewListShipmentTrackingEventsUseCase(db, repositories.NewShipmentStatusEventRepo(db)),
			appchannels.NewListBolShipmentConfirmationsUseCase(db, salesOrderRepo, repositories.NewBolShipmentConfirmationRepo(db)),
			search),
		tools.NewSearchCustomersTool(appcustomers.NewListCustomersUseCase(db, customerRepo)),
		tools.NewGetCustomerTool(
			appcustomers.NewGetCustomerUseCase(db, customerRepo),
			appcustomers.NewListContactPersonsUseCase(db, customerRepo, repositories.NewCustomerContactPersonRepo(db)),
			orders),
		tools.NewGetStockByWarehouseTool(
			apppms.NewGetProductUseCase(db, productRepo, repositories.NewVariantRepo(db), repositories.NewSKURepo(db)),
			appinventory.NewListWarehousesUseCase(db, repositories.NewWarehouseRepo(db)),
			appinventory.NewGetScopedStockUseCase(db, movementRepo),
			appinventory.NewGetStockLevelsUseCase(db, movementRepo, repositories.NewBatchRepo(db), repositories.NewTenantSettingsRepo(db))),
	}
}
