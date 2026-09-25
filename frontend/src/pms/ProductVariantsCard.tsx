import { FormCard } from './FormCard'
import { VariantsSection } from './VariantsSection'
import type { useProductDetail } from './hooks/useProductDetail'

interface ProductVariantsCardProps {
  detail: ReturnType<typeof useProductDetail>
  savedProductId: string | null
  variantsMode: boolean
  // Only offered when a WooCommerce store is connected.
  showWooVariationsToggle: boolean
  syncVariantsAsWooVariations: boolean
  onSyncVariantsAsWooVariationsChange: (value: boolean) => void
  enablingVariants: boolean
  enableVariantsError: string
  onEnableVariants: () => void
  onDisableVariants: () => void
}

// The "Variants" card: either the switch into variants mode, or (once on)
// the variants manager itself.
export function ProductVariantsCard({
  detail, savedProductId, variantsMode, showWooVariationsToggle, syncVariantsAsWooVariations, onSyncVariantsAsWooVariationsChange,
  enablingVariants, enableVariantsError, onEnableVariants, onDisableVariants,
}: ProductVariantsCardProps) {
  return (
    <FormCard title="Variants">
      {variantsMode ? (
        <>
          {showWooVariationsToggle && (
            <label className="flex items-start gap-2 mb-3 cursor-pointer">
              <input
                type="checkbox"
                checked={syncVariantsAsWooVariations}
                onChange={e => onSyncVariantsAsWooVariationsChange(e.target.checked)}
                className="mt-0.5"
              />
              <span className="text-2xs text-ink-3">
                <span className="font-medium text-ink-2">Sync variants as WooCommerce variations</span>
                <br />
                On: one Woo product with a variation per variant. Off (default): each variant becomes its own
                separate Woo product. Changing this after the product is already synced deletes the old
                WooCommerce listing(s) and recreates them in the new shape.
              </span>
            </label>
          )}
          {savedProductId ? (
            <VariantsSection detail={detail} productId={savedProductId} />
          ) : (
            <p className="text-2xs text-ink-4">Save the product first, then add variants below.</p>
          )}
          {!savedProductId && (
            <button
              type="button"
              onClick={onDisableVariants}
              className="text-2xs text-ink-4 hover:text-ink-2 transition-colors"
            >
              ← Actually, no variants
            </button>
          )}
        </>
      ) : (
        <div>
          <button
            type="button"
            onClick={onEnableVariants}
            disabled={enablingVariants}
            className="text-xs font-medium text-accent hover:text-accent/80 transition-colors disabled:opacity-50"
          >
            {enablingVariants ? 'Enabling…' : '+ Add variants like size or color'}
          </button>
          {enableVariantsError && <p className="text-2xs text-danger mt-1.5">{enableVariantsError}</p>}
        </div>
      )}
    </FormCard>
  )
}
