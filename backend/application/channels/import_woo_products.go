package channels

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"path"
	"strings"
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

// mediaUploader is the one method of application/pms's UploadMediaUseCase
// this use case needs (ISP) — same "call after the transaction commits,
// never nested inside it" reasoning as stockRecorder above, since Upload
// itself opens two transactions of its own around a storage write.
type mediaUploader interface {
	Execute(ctx context.Context, in apppms.UploadMediaInput) (*pmsDomain.Media, error)
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
	media            mediaUploader
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
	media mediaUploader,
) *ImportWooProductsUseCase {
	return &ImportWooProductsUseCase{
		db: db, connRepo: connRepo, wooFactory: wooFactory, encryptionSecret: encryptionSecret,
		categoryRepo: categoryRepo, productRepo: productRepo, variantRepo: variantRepo,
		skuRepo: skuRepo, listingRepo: listingRepo, stock: stock, media: media,
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

	// Fetched once per run, not per product — a product's own Categories
	// only carry {id, name, slug}, no parent, so reconstructing the tree
	// needs this separate, flat listing (see WooCategory's doc comment).
	cats, err := client.ListCategories(ctx)
	if err != nil {
		return nil, fmt.Errorf("importWooProducts: %w: %w", integrationsDomain.ErrWooRequestFailed, err)
	}
	catsByID := make(map[string]integrationsDomain.WooCategory, len(cats))
	for _, c := range cats {
		catsByID[c.ExternalID] = c
	}

	// Best-effort, not fatal: weight/dimensions are a nice-to-have, not
	// core to the import the way categories are — a units lookup failure
	// shouldn't abort importing 805 products over it. Empty units make
	// convertWooWeight/convertWooLength return nil (skip), same as before
	// this field existed at all.
	weightUnit, dimensionUnit, err := client.GetProductUnits(ctx)
	if err != nil {
		weightUnit, dimensionUnit = "", ""
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
			uc.importOne(ctx, in, client, catsByID, weightUnit, dimensionUnit, wp, &result)
		}
	}
	return &result, nil
}

// importOne processes a single Woo product. Failures here are recorded on
// the result and never abort the rest of the import — one bad product
// (unparseable price, transient error) shouldn't block the other 199.
func (uc *ImportWooProductsUseCase) importOne(
	ctx context.Context, in ImportWooProductsInput, client integrationsDomain.WooClient,
	catsByID map[string]integrationsDomain.WooCategory, weightUnit, dimensionUnit string,
	wp integrationsDomain.WooProduct, result *ImportResult,
) {
	if !ShouldImportWooProductType(wp.Type) {
		result.SkippedType++
		return
	}

	var createdProductID, createdSKUID string
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

		priceMinorUnits, err := ParseWooPriceToMinorUnits(wp.RegularPrice)
		if err != nil {
			return err
		}

		now := time.Now()
		tenantID := tenant.IDFromContext(ctx)

		// A product's Woo category (and its ancestors, to preserve Woo's
		// hierarchy) is created in CommerceOS if it doesn't already exist
		// (matched by slug) — previously this only ever linked to a
		// pre-existing category, so any category Woo had that CommerceOS
		// didn't silently dropped and the product imported with no
		// category at all. pms.Product supports one category, so only the
		// first of Woo's (possibly several) categories is used; that
		// leaf's ParentID chain is what carries the hierarchy.
		var categoryID *string
		if len(wp.Categories) > 0 {
			id, err := uc.resolveCategoryChain(ctx, tenantID, now, catsByID, wp.Categories[0].ExternalID)
			if err != nil {
				return err
			}
			categoryID = id
		}

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
			IsVirtual: wp.Virtual,
			Status:    pmsDomain.SKUStatusActive, CreatedAt: now, UpdatedAt: now,
		}
		// A virtual SKU never ships, so it can never carry weight/dimensions
		// (enforced elsewhere too — see ErrVirtualSKUCannotHaveDimensions).
		if !wp.Virtual {
			sku.WeightGrams = convertWooWeight(wp.Weight, weightUnit)
			sku.LengthMM = convertWooLength(wp.Dimensions.Length, dimensionUnit)
			sku.WidthMM = convertWooLength(wp.Dimensions.Width, dimensionUnit)
			sku.HeightMM = convertWooLength(wp.Dimensions.Height, dimensionUnit)
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

		createdProductID = product.ID
		createdSKUID = sku.ID
		imported = true
		result.Imported++
		return nil
	})
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("product %s (%s): %v", wp.ID, wp.Name, err))
		return
	}
	if !imported {
		return
	}

	if wp.ManageStock && wp.StockQuantity != nil && *wp.StockQuantity != 0 {
		// Adjustment, not Receipt: this is recording Woo's reported stock
		// level as the new starting point, not a literal goods-receipt
		// event — Receipt requires a non-negative delta and Woo can report
		// negative stock (stores with overselling allowed), same reasoning
		// ImportWooStockLevelsUseCase already uses for its own reconciliation.
		_, err := uc.stock.Execute(ctx, invapp.RecordStockMovementInput{
			SKUID: createdSKUID, WarehouseID: in.DefaultWarehouseID,
			Type: invDomain.MovementTypeAdjustment, QuantityDelta: int64(*wp.StockQuantity),
			Reason: "woocommerce import", OccurredAt: time.Now(),
		})
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("product %s: initial stock: %v", wp.ID, err))
		}
	}

	uc.importImages(ctx, client, createdProductID, wp, result)
}

// resolveCategoryChain ensures every ancestor of externalID exists in
// CommerceOS (creating any that don't, root-first, so a child's ParentID
// always references an already-persisted row) and returns the leaf's
// CommerceOS id. Matching is by slug, same as before — two different Woo
// categories that happen to slugify to the same string (e.g. same name
// under different parents) still conflate into one CommerceOS category;
// that's a pre-existing limitation of the category model, not new here.
func (uc *ImportWooProductsUseCase) resolveCategoryChain(
	ctx context.Context, tenantID string, now time.Time,
	catsByID map[string]integrationsDomain.WooCategory, externalID string,
) (*string, error) {
	woo, ok := catsByID[externalID]
	if !ok {
		// Category list didn't include it (e.g. deleted between the
		// categories call and the products call) — degrade to
		// uncategorized rather than failing the whole product.
		return nil, nil
	}

	if cat, err := uc.categoryRepo.FindBySlug(ctx, woo.Slug); err == nil {
		return &cat.ID, nil
	} else if !errors.Is(err, pmsDomain.ErrCategoryNotFound) {
		return nil, err
	}

	var parentID *string
	if woo.ParentExternalID != "" {
		id, err := uc.resolveCategoryChain(ctx, tenantID, now, catsByID, woo.ParentExternalID)
		if err != nil {
			return nil, err
		}
		parentID = id
	}

	newCategory := pmsDomain.Category{
		TenantID: tenantID, ParentID: parentID, Name: woo.Name, Slug: woo.Slug,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := uc.categoryRepo.Create(ctx, &newCategory); err != nil {
		return nil, err
	}
	return &newCategory.ID, nil
}

// importImages downloads each Woo image and re-uploads it through our own
// FileStorage rather than just linking Woo's URL — the product stays
// viewable if the store connection is later removed, and we're not
// hotlinking another site's media indefinitely. Runs after the product's
// own transaction has committed (mediaUploader opens its own transactions
// around a storage write — see mediaUploader's doc comment). Per-image
// failures are recorded and never abort the rest of the product's images
// or the run.
func (uc *ImportWooProductsUseCase) importImages(ctx context.Context, client integrationsDomain.WooClient, productID string, wp integrationsDomain.WooProduct, result *ImportResult) {
	for _, img := range wp.Images {
		if img.URL == "" {
			continue
		}
		data, contentType, err := client.DownloadImage(ctx, img.URL)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("product %s: image %s: %v", wp.ID, img.URL, err))
			continue
		}
		contentType = strings.TrimSpace(strings.SplitN(contentType, ";", 2)[0])
		_, err = uc.media.Execute(ctx, apppms.UploadMediaInput{
			ProductID: &productID, Filename: filenameFromURL(img.URL), ContentType: contentType, Data: data, AltText: img.Alt,
		})
		data.Close()
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("product %s: image %s: %v", wp.ID, img.URL, err))
		}
	}
}

func filenameFromURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "image"
	}
	name := path.Base(u.Path)
	if name == "" || name == "." || name == "/" {
		return "image"
	}
	return name
}
