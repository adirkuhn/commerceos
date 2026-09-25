package channels

import (
	"context"
	"errors"
	"log"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
)

// createProductSelfHealingDuplicateSKU wraps client.CreateProduct --
// WooCommerce rejects a create when a product with this SKU already
// exists on the store (its own SKU-uniqueness constraint); in production
// that happens when this codebase's own link to an already-existing Woo
// product was lost (e.g. a connection disconnect/reconnect orphaned the
// product_channel_listings row, see integrationsDomain.ErrWooDuplicateSKU's
// doc), not because the product is actually new. Rather than failing this
// push forever on every retry, look the existing product up by SKU and
// treat it as if that had just been created -- re-links CommerceOS to the
// product Woo already has instead of endlessly retrying a create Woo will
// always reject. If FindProductBySKU alone comes up empty -- confirmed in
// production against a real conflicting SKU that genuinely existed on the
// store -- selfHealFromTrash below covers the one concrete case that
// explains: the conflicting product sitting in WordPress's trash, which
// "any" status search excludes even though its SKU is still reserved.
// Only wired into the two CreateProduct call sites (single-SKU and
// separate-products modes) that actually hit this failure in practice;
// CreateVariation (variations mode) isn't covered.
func (uc *PushProductToWooUseCase) createProductSelfHealingDuplicateSKU(ctx context.Context, client integrationsDomain.WooClient, wooProduct integrationsDomain.WooProduct) (*integrationsDomain.WooProduct, error) {
	created, err := client.CreateProduct(ctx, wooProduct)
	if err == nil {
		return created, nil
	}
	if !errors.Is(err, integrationsDomain.ErrWooDuplicateSKU) {
		return nil, err
	}
	found, findErr := client.FindProductBySKU(ctx, wooProduct.SKU)
	if findErr != nil {
		log.Printf("createProductSelfHealingDuplicateSKU: FindProductBySKU(%q) errored, self-heal miss: %v", wooProduct.SKU, findErr)
		return nil, err // self-heal didn't pan out -- surface the original error
	}
	if found != nil {
		log.Printf("createProductSelfHealingDuplicateSKU: FindProductBySKU(%q) found existing Woo product %s, self-healing", wooProduct.SKU, found.ID)
		return found, nil
	}
	log.Printf("createProductSelfHealingDuplicateSKU: FindProductBySKU(%q) found nothing among normal products, checking trash", wooProduct.SKU)
	return uc.selfHealFromTrash(ctx, client, wooProduct, err)
}

// selfHealFromTrash handles the case FindProductBySKU alone can't:
// WordPress's own "any" status excludes trash, so a product moved to trash
// still reserves its SKU (blocking the create that got us here) while
// being invisible to the normal search above. If the conflicting product
// turns out to be sitting in trash, purge it for real (same cleanup a
// store admin does by hand from wp-admin) and retry the original create
// once -- origErr is what surfaces if trash search/delete/retry don't pan
// out either, so the caller still sees the original duplicate-SKU failure
// rather than a confusing secondary error.
func (uc *PushProductToWooUseCase) selfHealFromTrash(ctx context.Context, client integrationsDomain.WooClient, wooProduct integrationsDomain.WooProduct, origErr error) (*integrationsDomain.WooProduct, error) {
	trashed, trashErr := client.FindTrashedProductBySKU(ctx, wooProduct.SKU)
	if trashErr != nil {
		log.Printf("createProductSelfHealingDuplicateSKU: FindTrashedProductBySKU(%q) errored, self-heal miss: %v", wooProduct.SKU, trashErr)
		return nil, origErr
	}
	if trashed == nil {
		log.Printf("createProductSelfHealingDuplicateSKU: FindTrashedProductBySKU(%q) found nothing either, self-heal miss", wooProduct.SKU)
		return nil, origErr
	}
	log.Printf("createProductSelfHealingDuplicateSKU: found trashed Woo product %s holding SKU %q, deleting it and retrying create", trashed.ID, wooProduct.SKU)
	if delErr := client.DeleteProduct(ctx, trashed.ID); delErr != nil {
		log.Printf("createProductSelfHealingDuplicateSKU: failed to delete trashed product %s, self-heal miss: %v", trashed.ID, delErr)
		return nil, origErr
	}
	recreated, retryErr := client.CreateProduct(ctx, wooProduct)
	if retryErr != nil {
		log.Printf("createProductSelfHealingDuplicateSKU: retry create after deleting trashed product %s still failed: %v", trashed.ID, retryErr)
		return nil, origErr
	}
	return recreated, nil
}
