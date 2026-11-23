package assistanttools

import (
	"time"

	"gorm.io/gorm"

	appassistant "github.com/adirkuhn/commerceos/application/assistant"
	"github.com/adirkuhn/commerceos/application/assistant/actions"
	appcustomers "github.com/adirkuhn/commerceos/application/customers"
	appinventory "github.com/adirkuhn/commerceos/application/inventory"
	appquotes "github.com/adirkuhn/commerceos/application/quotes"
	"github.com/adirkuhn/commerceos/domain/outbox"
	omshandler "github.com/adirkuhn/commerceos/infrastructure/handlers/oms"
	pmshandler "github.com/adirkuhn/commerceos/infrastructure/handlers/pms"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
)

// ProposalActions builds every change the assistant can propose, on the
// same use cases the quotes admin uses (see handlers/quotes.Register). db
// is the app's RLS-enforced connection; outboxRepo records the events the
// quote and order use cases raise when a proposal is confirmed.
func ProposalActions(db *gorm.DB, outboxRepo outbox.Repository) []appassistant.ProposalAction {
	quoteRepo := repositories.NewQuoteRepo(db)
	customers := appcustomers.NewGetCustomerUseCase(db, repositories.NewCustomerRepo(db))
	priceList := pmshandler.NewListPriceListUseCase(db)

	return []appassistant.ProposalAction{
		actions.NewCreateQuoteAction(priceList, customers,
			appquotes.NewCreateQuoteUseCase(db, quoteRepo, customers, outboxRepo), time.Now),
		actions.NewConvertQuoteAction(priceList,
			appquotes.NewGetQuoteUseCase(db, quoteRepo),
			appinventory.NewListWarehousesUseCase(db, repositories.NewWarehouseRepo(db)),
			appquotes.NewCheckQuoteStockUseCase(db, quoteRepo,
				appinventory.NewAllocateSaleStockUseCase(db, repositories.NewStockMovementRepo(db), repositories.NewBatchRepo(db), outboxRepo)),
			appquotes.NewConvertQuoteToSalesOrderUseCase(db, appquotes.ConvertQuoteDeps{
				Quotes: quoteRepo, Catalog: priceList, Customers: customers,
				Orders: omshandler.NewCreateSalesOrderUseCase(db, outboxRepo), Outbox: outboxRepo, Now: time.Now,
			}),
		),
	}
}
