package search

import (
	"context"
	"fmt"
	"strings"

	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	suppliersDomain "github.com/adirkuhn/commerceos/domain/suppliers"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// perModuleLimit bounds how many results each entity type contributes --
// this is a Cmd+K quick-jump, not a full search results page, so a handful
// of best matches per module is all the UI needs to show.
const perModuleLimit = 5

// GlobalSearchResult is Cmd+K's read-model -- every entity type's own
// domain struct returned as-is (no DTO flattening here; the API handler
// maps to the OpenAPI response shape), grouped so the frontend can render
// one section per module. Only tenant-scoped entities are searched --
// there is no local user directory in this backend (auth is external, via
// lm2), so a "Users" section is deliberately absent.
type GlobalSearchResult struct {
	Products    []pmsDomain.Product
	Suppliers   []suppliersDomain.Supplier
	Customers   []customersDomain.Customer
	SalesOrders []omsDomain.SalesOrder
}

// GlobalSearchUseCase composes each domain's own existing search method
// into one call -- mirrors application/dashboard.GetDashboardStatsUseCase's
// pattern of a cross-domain read use case depending directly on other
// domains' repository ports (never their use cases, never their raw
// tables) rather than folding this into any one domain. Products reuse
// ProductRepository.List's existing Search filter (already
// accent-insensitive + typo-tolerant, see pms.ProductFilter.Search's doc);
// Suppliers/Customers/SalesOrders use the Search method added alongside
// this feature (127_search_trigram_indexes.sql), same matching convention.
type GlobalSearchUseCase struct {
	db             *gorm.DB
	productRepo    pmsDomain.ProductRepository
	supplierRepo   suppliersDomain.SupplierRepository
	customerRepo   customersDomain.CustomerRepository
	salesOrderRepo omsDomain.SalesOrderRepository
}

func NewGlobalSearchUseCase(
	db *gorm.DB,
	productRepo pmsDomain.ProductRepository,
	supplierRepo suppliersDomain.SupplierRepository,
	customerRepo customersDomain.CustomerRepository,
	salesOrderRepo omsDomain.SalesOrderRepository,
) *GlobalSearchUseCase {
	return &GlobalSearchUseCase{
		db: db, productRepo: productRepo, supplierRepo: supplierRepo,
		customerRepo: customerRepo, salesOrderRepo: salesOrderRepo,
	}
}

func (uc *GlobalSearchUseCase) Execute(ctx context.Context, query string) (*GlobalSearchResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return &GlobalSearchResult{}, nil
	}

	result := &GlobalSearchResult{}
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		products, _, err := uc.productRepo.List(ctx, pmsDomain.ProductFilter{Search: &query, Limit: perModuleLimit})
		if err != nil {
			return fmt.Errorf("search products: %w", err)
		}
		result.Products = products

		suppliers, err := uc.supplierRepo.Search(ctx, query, perModuleLimit)
		if err != nil {
			return fmt.Errorf("search suppliers: %w", err)
		}
		result.Suppliers = suppliers

		customers, err := uc.customerRepo.Search(ctx, query, perModuleLimit)
		if err != nil {
			return fmt.Errorf("search customers: %w", err)
		}
		result.Customers = customers

		salesOrders, err := uc.salesOrderRepo.Search(ctx, query, perModuleLimit)
		if err != nil {
			return fmt.Errorf("search sales orders: %w", err)
		}
		result.SalesOrders = salesOrders
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("globalSearch: %w", err)
	}
	return result, nil
}
