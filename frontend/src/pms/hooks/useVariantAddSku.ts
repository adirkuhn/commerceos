import i18next from 'i18next'
import { useState } from 'react'
import type { TenantSettings } from '../../settings/types'
import type { CreateSKUPayload } from '../types'
import { emptyVariantSkuDraft, variantSkuAddPayload, type VariantSkuDraft } from '../variantSkuForms'

export type SetVariantSkuDraftField = <K extends keyof VariantSkuDraft>(key: K, value: VariantSkuDraft[K]) => void
export type PatchVariantSkuDraft = (patch: Partial<VariantSkuDraft>) => void

// State + submit for a variant row's "+ Add SKU" panel. Lives with the row
// (not the panel) so typed values survive Cancel and reopen; only a
// successful add resets them.
export function useVariantAddSku(
  variantId: string,
  addSKU: (variantId: string, payload: CreateSKUPayload) => Promise<void>,
  tenantSettings: TenantSettings | null,
) {
  const [open, setOpen] = useState(false)
  const [draft, setDraft] = useState<VariantSkuDraft>(() => emptyVariantSkuDraft('USD'))
  const setField: SetVariantSkuDraftField = (key, value) => setDraft(prev => ({ ...prev, [key]: value }))
  const patch: PatchVariantSkuDraft = p => setDraft(prev => ({ ...prev, ...p }))
  const [portionPickerOpen, setPortionPickerOpen] = useState(false)
  const [error, setError] = useState('')
  const [submitting, setSubmitting] = useState(false)

  // Seed the currency from the tenant's default once settings actually
  // arrive (not just !loading, which is false for an instant before the
  // fetch starts).
  const [currencyDefaulted, setCurrencyDefaulted] = useState(false)
  if (!currencyDefaulted && tenantSettings) {
    if (tenantSettings.default_currency) setField('currency', tenantSettings.default_currency)
    setCurrencyDefaulted(true)
  }

  const submit = async () => {
    setError('')
    setSubmitting(true)
    try {
      await addSKU(variantId, variantSkuAddPayload(draft))
      setOpen(false)
      setDraft(emptyVariantSkuDraft(tenantSettings?.default_currency || 'USD'))
    } catch (err) {
      setError(err instanceof Error ? err.message : i18next.t('pms:errors.addSku'))
    } finally {
      setSubmitting(false)
    }
  }

  return { open, setOpen, draft, setField, patch, portionPickerOpen, setPortionPickerOpen, error, submitting, submit }
}
