// Package assistanttools wires the assistant's read-only tools
// (application/assistant/tools) to the real read use cases and
// repositories -- the one place that knows the concrete adapters.
package assistanttools

import (
	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/application/assistant/tools"
	appchannels "github.com/adirkuhn/commerceos/application/channels"
	appintegrations "github.com/adirkuhn/commerceos/application/integrations"
	appinventory "github.com/adirkuhn/commerceos/application/inventory"
	appoms "github.com/adirkuhn/commerceos/application/oms"
	apppms "github.com/adirkuhn/commerceos/application/pms"
	appquotes "github.com/adirkuhn/commerceos/application/quotes"
	appsearch "github.com/adirkuhn/commerceos/application/search"
	appsettings "github.com/adirkuhn/commerceos/application/settings"
	apptax "github.com/adirkuhn/commerceos/application/tax"
	"github.com/adirkuhn/commerceos/domain/assistant"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
)

// ReadTools builds every read-only assistant tool. db is the app's
// (RLS-enforced) connection: each use case opens its own tenant.WithTenant
// transaction from the ctx the tool is called with, so the tools only ever
// see what the requesting user's tenant holds.
func ReadTools(db *gorm.DB) []assistant.Tool {
	productRepo := repositories.NewProductRepo(db)
	salesOrderRepo := repositories.NewSalesOrderRepo(db)
	settingsRepo := repositories.NewTenantSettingsRepo(db)
	quoteRepo := repositories.NewQuoteRepo(db)
	salesChannels := appchannels.NewListSalesChannelsUseCase(db, repositories.NewSalesChannelRepo(db))

	search := appsearch.NewGlobalSearchUseCase(db, productRepo, repositories.NewSupplierRepo(db),
		repositories.NewCustomerRepo(db), salesOrderRepo)

	core := []assistant.Tool{
		tools.NewSearchTool(search),
		tools.NewGetSalesOrderTool(
			appoms.NewGetSalesOrderUseCase(db, salesOrderRepo, repositories.NewSalesOrderLineItemRepo(db), repositories.NewSalesOrderPaymentRepo(db)),
			search).WithOrigins(appintegrations.NewListSalesOrderOriginsUseCase(db, repositories.NewSalesOrderOriginRepo(db))),
		tools.NewListSalesOrdersTool(appoms.NewListSalesOrdersUseCase(db, salesOrderRepo, settingsRepo)),
		tools.NewGetQuoteTool(appquotes.NewGetQuoteUseCase(db, quoteRepo), appquotes.NewListQuotesUseCase(db, quoteRepo)),
		tools.NewListQuotesTool(appquotes.NewListQuotesUseCase(db, quoteRepo)),
		tools.NewGetProductTool(
			apppms.NewGetProductUseCase(db, productRepo, repositories.NewVariantRepo(db), repositories.NewSKURepo(db)),
			appinventory.NewGetStockLevelsUseCase(db, repositories.NewStockMovementRepo(db), repositories.NewBatchRepo(db), settingsRepo)),
		tools.NewListSalesChannelsTool(salesChannels),
		tools.NewListProductChannelListingsTool(salesChannels,
			appchannels.NewListProductChannelListingsUseCase(db, repositories.NewProductChannelListingRepo(db))),
		tools.NewGetTenantSettingsTool(appsettings.NewGetTenantSettingsUseCase(db, settingsRepo)),
		tools.NewGetCompanyProfileTool(appsettings.NewGetCompanyProfileUseCase(db, repositories.NewCompanyProfileRepo(db))),
		tools.NewListTaxRatesTool(
			apptax.NewListTaxClassesUseCase(db, repositories.NewTaxClassRepo(db)),
			apptax.NewListTaxRatesUseCase(db, repositories.NewTaxRateRepo(db))),
	}
	core = append(core, salesTools(db)...)
	return append(core, operationsTools(db)...)
}
