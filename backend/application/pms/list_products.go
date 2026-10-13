package pms

import (
	"context"
	"fmt"
	"time"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

const (
	defaultProductsPerPage = 20
	maxProductsPerPage     = 200
)

type ListProductsInput struct {
	Page         int
	PerPage      int
	Search       string
	UpdatedAfter *time.Time
	// AndCategoryID/CategoryID/Tag/IncludeIDs/ExcludeIDs pass straight
	// through to pmsDomain.ProductFilter's fields of the same name --
	// application/storefront is the caller that populates these (see that
	// package's own doc comment for why: this use case stays enrollment-
	// agnostic, storefront translates a channel's actual rule into these
	// generic filter primitives).
	AndCategoryID *string
	AndIncludeIDs *[]string
	CategoryID    *string
	Tag           *string
	IncludeIDs    *[]string
	ExcludeIDs    []string
	// Status restricts to one product status -- nil (the admin catalog's
	// default) means every status. application/storefront always passes
	// ProductStatusActive: a Draft/Archived product must never render on a
	// live storefront, same rule pms.Product.IsSyncableToChannels enforces
	// for external channel pushes.
	Status *pmsDomain.ProductStatus
	// OrderByIDs -- see pmsDomain.ProductFilter.OrderByIDs.
	OrderByIDs []string
	// StockLevel -- see pmsDomain.ProductFilter.StockLevel.
	StockLevel pmsDomain.StockLevel
}

type ListProductsResult struct {
	Products []pmsDomain.Product
	Total    int64
}

type ListProductsUseCase struct {
	db          *gorm.DB
	productRepo pmsDomain.ProductRepository
}

func NewListProductsUseCase(db *gorm.DB, productRepo pmsDomain.ProductRepository) *ListProductsUseCase {
	return &ListProductsUseCase{db: db, productRepo: productRepo}
}

func (uc *ListProductsUseCase) Execute(ctx context.Context, in ListProductsInput) (*ListProductsResult, error) {
	page := in.Page
	if page <= 0 {
		page = 1
	}
	perPage := in.PerPage
	if perPage <= 0 {
		perPage = defaultProductsPerPage
	}
	if perPage > maxProductsPerPage {
		perPage = maxProductsPerPage
	}

	filter := pmsDomain.ProductFilter{
		Limit: perPage, Offset: (page - 1) * perPage,
		AndCategoryID: in.AndCategoryID, AndIncludeIDs: in.AndIncludeIDs, CategoryID: in.CategoryID, Tag: in.Tag,
		IncludeIDs: in.IncludeIDs, ExcludeIDs: in.ExcludeIDs, Status: in.Status, OrderByIDs: in.OrderByIDs,
		StockLevel: in.StockLevel,
	}
	if in.Search != "" {
		filter.Search = &in.Search
	}
	if in.UpdatedAfter != nil {
		filter.UpdatedAfter = in.UpdatedAfter
	}

	result := &ListProductsResult{}
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		p, total, err := uc.productRepo.List(ctx, filter)
		if err != nil {
			return err
		}
		result.Products = p
		result.Total = total
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("listProducts: %w", err)
	}
	return result, nil
}
