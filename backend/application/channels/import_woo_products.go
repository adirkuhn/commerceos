package channels

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	invapp "github.com/adirkuhn/commerceos/application/inventory"
	apppms "github.com/adirkuhn/commerceos/application/pms"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/crypto"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// stockRecorder is the one method of application/inventory's
// RecordStockMovementUseCase this use case needs — defined at point of use
// (ISP) so tests can stub it without a real DB. Deliberately called as a
// separate top-level operation *after* a product's own tenant.WithTenant
// transaction commits, never nested inside it: tenant.WithTenant opens a
// real DB transaction via the shared *gorm.DB, and GORM has no way to
// detect "there's already an open transaction in this ctx" unless you call
// .Transaction() on the tx object itself — calling another use case's
// Execute (which calls tenant.WithTenant again on the same top-level *gorm.DB)
// from inside an outer transaction's closure would silently open a second,
// unrelated transaction rather than nesting.
type stockRecorder interface {
	Execute(ctx context.Context, in invapp.RecordStockMovementInput) (*invDomain.StockMovement, error)
}

const maxImportPages = 100 // safety cap against a runaway pagination loop

type ImportWooProductsInput struct {
	ConnectionID       string
	DefaultWarehouseID string // where imported stock is attributed to — Woo has no warehouse concept
	DefaultCurrency    string // Woo's product endpoint doesn't return currency; it's a store-level setting
}

type ImportResult struct {
	Imported      int
	AlreadyLinked int
	SkippedType   int
	Errors        []string
}

type ImportWooProductsUseCase struct {
	db               *gorm.DB
	connRepo         integrationsDomain.ConnectionRepository
	wooFactory       integrationsDomain.WooClientFactory
	encryptionSecret string
	categoryRepo     pmsDomain.CategoryRepository
	productRepo      pmsDomain.ProductRepository
	variantRepo      pmsDomain.VariantRepository
	skuRepo          pmsDomain.SKURepository
	listingRepo      channelsDomain.ProductChannelListingRepository
	stock            stockRecorder
}

func NewImportWooProductsUseCase(
	db *gorm.DB,
	connRepo integrationsDomain.ConnectionRepository,
	wooFactory integrationsDomain.WooClientFactory,
	encryptionSecret string,
	categoryRepo pmsDomain.CategoryRepository,
	productRepo pmsDomain.ProductRepository,
	variantRepo pmsDomain.VariantRepository,
	skuRepo pmsDomain.SKURepository,
	listingRepo channelsDomain.ProductChannelListingRepository,
	stock stockRecorder,
) *ImportWooProductsUseCase {
	return &ImportWooProductsUseCase{
		db: db, connRepo: connRepo, wooFactory: wooFactory, encryptionSecret: encryptionSecret,
		categoryRepo: categoryRepo, productRepo: productRepo, variantRepo: variantRepo,
		skuRepo: skuRepo, listingRepo: listingRepo, stock: stock,
	}
}

func (uc *ImportWooProductsUseCase) Execute(ctx context.Context, in ImportWooProductsInput) (*ImportResult, error) {
	var conn integrationsDomain.Connection
	var secret string
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		existing, err := uc.connRepo.FindByID(ctx, in.ConnectionID)
		if err != nil {
			return err
		}
		conn = *existing
		secret, err = crypto.Decrypt(uc.encryptionSecret, conn.ConsumerSecretEncrypted)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("importWooProducts: %w", err)
	}

	client, err := uc.wooFactory.NewClient(conn.StoreURL.String(), conn.ConsumerKey, secret)
	if err != nil {
		return nil, fmt.Errorf("importWooProducts: %w: %w", integrationsDomain.ErrWooRequestFailed, err)
	}

	result := ImportResult{}
	for page := 1; page <= maxImportPages; page++ {
		products, err := client.ListProducts(ctx, page, 50)
		if err != nil {
			return nil, fmt.Errorf("importWooProducts: %w: %w", integrationsDomain.ErrWooRequestFailed, err)
		}
		if len(products) == 0 {
			break
		}
		for _, wp := range products {
			uc.importOne(ctx, in, wp, &result)
		}
	}
	return &result, nil
}

// importOne processes a single Woo product. Failures here are recorded on
// the result and never abort the rest of the import — one bad product
// (unparseable price, transient error) shouldn't block the other 199.
func (uc *ImportWooProductsUseCase) importOne(ctx context.Context, in ImportWooProductsInput, wp integrationsDomain.WooProduct, result *ImportResult) {
	if !ShouldImportWooProductType(wp.Type) {
		result.SkippedType++
		return
	}

	var createdSKUID string
	var imported bool
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		_, err := uc.listingRepo.FindByExternalID(ctx, in.ConnectionID, wp.ID)
		if err == nil {
			result.AlreadyLinked++
			return nil
		}
		if !errors.Is(err, channelsDomain.ErrListingNotFound) {
			return err
		}

		var categoryID *string
		if len(wp.Categories) > 0 {
			if cat, err := uc.categoryRepo.FindBySlug(ctx, wp.Categories[0].Slug); err == nil {
				categoryID = &cat.ID
			}
		}

		priceMinorUnits, err := ParseWooPriceToMinorUnits(wp.RegularPrice)
		if err != nil {
			return err
		}

		now := time.Now()
		tenantID := tenant.IDFromContext(ctx)
		slug := apppms.Slugify(wp.Name)
		if _, err := uc.productRepo.FindBySlug(ctx, slug); err == nil {
			slug = slug + "-" + wp.ID // collision with an unrelated existing product
		}

		product := pmsDomain.Product{
			TenantID: tenantID, CategoryID: categoryID, Name: wp.Name, Slug: slug,
			Description: wp.Description, Status: MapWooStatus(wp.Status), HasVariants: false,
			CreatedAt: now, UpdatedAt: now,
		}
		if err := uc.productRepo.Create(ctx, &product); err != nil {
			return err
		}

		variant := pmsDomain.Variant{
			TenantID: tenantID, ProductID: product.ID, Name: product.Name, Position: 0,
			CreatedAt: now, UpdatedAt: now,
		}
		if err := uc.variantRepo.Create(ctx, &variant); err != nil {
			return err
		}

		skuCode := wp.SKU
		if skuCode == "" {
			skuCode = "WOO-" + wp.ID
		}
		sku := pmsDomain.SKU{
			TenantID: tenantID, VariantID: variant.ID, SKUCode: skuCode,
			PriceMinorUnits: priceMinorUnits, CurrencyCode: in.DefaultCurrency,
			IsVirtual: wp.Virtual, // weight/dimensions stay unimported regardless (unit system unknown)
			Status:    pmsDomain.SKUStatusActive, CreatedAt: now, UpdatedAt: now,
		}
		if err := uc.skuRepo.Create(ctx, &sku); err != nil {
			return err
		}

		listing := channelsDomain.ProductChannelListing{
			TenantID: tenantID, ProductID: product.ID, ConnectionID: in.ConnectionID,
			Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: wp.ID,
			SyncStatus: channelsDomain.SyncStatusSynced, LastSyncedAt: &now,
			CreatedAt: now, UpdatedAt: now,
		}
		if err := uc.listingRepo.Create(ctx, &listing); err != nil {
			return err
		}

		createdSKUID = sku.ID
		imported = true
		result.Imported++
		return nil
	})
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("product %s (%s): %v", wp.ID, wp.Name, err))
		return
	}

	if imported && wp.ManageStock && wp.StockQuantity != nil && *wp.StockQuantity != 0 {
		_, err := uc.stock.Execute(ctx, invapp.RecordStockMovementInput{
			SKUID: createdSKUID, WarehouseID: in.DefaultWarehouseID,
			Type: invDomain.MovementTypeReceipt, QuantityDelta: int64(*wp.StockQuantity),
			Reason: "woocommerce import", OccurredAt: time.Now(),
		})
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("product %s: initial stock: %v", wp.ID, err))
		}
	}
}
