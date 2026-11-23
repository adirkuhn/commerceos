package assistanttools

import (
	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/application/assistant/tools"
	appchannels "github.com/adirkuhn/commerceos/application/channels"
	appintegrations "github.com/adirkuhn/commerceos/application/integrations"
	appinventory "github.com/adirkuhn/commerceos/application/inventory"
	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	appsuppliers "github.com/adirkuhn/commerceos/application/suppliers"
	"github.com/adirkuhn/commerceos/domain/assistant"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
)

// operationsTools: suppliers, purchase orders, batches and expiry,
// integration connections and sync logs -- wired like the suppliers,
// procurement, inventory and integrations handlers do.
func operationsTools(db *gorm.DB) []assistant.Tool {
	suppliers := appsuppliers.NewListSuppliersUseCase(db, repositories.NewSupplierRepo(db))
	poRepo, lineItemRepo := repositories.NewPurchaseOrderRepo(db), repositories.NewPurchaseOrderLineItemRepo(db)
	purchaseOrders := appprocurement.NewListPurchaseOrdersUseCase(db, poRepo, lineItemRepo)
	warehouseRepo := repositories.NewWarehouseRepo(db)
	warehouses := appinventory.NewListWarehousesUseCase(db, warehouseRepo)
	batchRepo := repositories.NewBatchRepo(db)
	syncLogRepo, productRepo, connRepo := repositories.NewSyncLogRepo(db), repositories.NewProductRepo(db), repositories.NewConnectionRepo(db)

	return []assistant.Tool{
		tools.NewSearchSuppliersTool(suppliers),
		tools.NewGetSupplierTool(suppliers, purchaseOrders),
		tools.NewListPurchaseOrdersTool(purchaseOrders, suppliers),
		tools.NewGetPurchaseOrderTool(
			appprocurement.NewGetPurchaseOrderUseCase(db, poRepo, lineItemRepo, repositories.NewPurchaseOrderDocumentRepo(db),
				repositories.NewSKURepo(db), repositories.NewVariantRepo(db), productRepo),
			suppliers, warehouses),
		tools.NewListBatchesTool(
			appinventory.NewListBatchesUseCase(db, batchRepo, repositories.NewStockMovementRepo(db), warehouseRepo), warehouses),
		tools.NewListExpiringStockTool(appinventory.NewListExpiringBatchesUseCase(db, batchRepo)),
		tools.NewListIntegrationConnectionsTool(
			appintegrations.NewListConnectionsUseCase(db, connRepo),
			appintegrations.NewGetAIConnectionUseCase(db, repositories.NewAIConnectionRepo(db))),
		tools.NewListSyncLogsTool(
			appchannels.NewListAllSyncLogsUseCase(db, syncLogRepo, productRepo, connRepo),
			appchannels.NewListSyncLogsNeedingAttentionUseCase(db, syncLogRepo, productRepo, connRepo)),
	}
}
