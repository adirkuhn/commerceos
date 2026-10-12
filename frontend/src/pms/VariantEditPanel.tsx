import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { SKUCurrencyPricesEditor } from './SkuAliasEditors'
import { CURRENCIES } from './productFormat'
import { fieldInputClass, fieldLabelClass } from './formStyles'
import { variantEditFormFrom, variantEditSkuPayload, type VariantEditForm } from './variantSkuForms'
import { DimensionInputsGrid } from './DimensionInputsGrid'
import type { SKU, SKUStatus, UpdateSKUPayload, UpdateVariantPayload, VariantWithSKUs } from './types'
import { apiErrorText } from '../api/apiErrorText'

interface VariantEditPanelProps {
  variant: VariantWithSKUs
  sku: SKU | undefined
  updateVariant: (id: string, payload: UpdateVariantPayload) => Promise<void>
  updateSKU: (id: string, payload: UpdateSKUPayload) => Promise<void>
  onClose: () => void
}

// Editing an existing variant/SKU's own fields (name, price, status,
// weight/dimensions, etc.) -- everything the "Add SKU" form covers except
// barcode and the weight-pool link, which already have their own dedicated
// inline-edit controls; adding a second path to change those here would
// just create two conflicting places to do the same thing. Saves both the
// variant (name) and the SKU (everything else) as two separate calls --
// they're different entities/endpoints already.
//
// Prefilled from the current variant/SKU on mount -- the row remounts it
// (via key) on every "Edit" click, discarding unsaved edits.
export function VariantEditPanel({ variant, sku, updateVariant, updateSKU, onClose }: VariantEditPanelProps) {
  const [form, setForm] = useState<VariantEditForm>(() => variantEditFormFrom(variant, sku))
  const set = <K extends keyof VariantEditForm>(key: K, value: VariantEditForm[K]) => setForm(prev => ({ ...prev, [key]: value }))
  const { t } = useTranslation(['pms', 'common'])
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)

  const save = async () => {
    setError('')
    setSaving(true)
    try {
      await updateVariant(variant.id, {
        name: form.name,
        position: variant.position,
        featured_media_id: variant.featured_media_id,
      })
      if (sku) await updateSKU(sku.id, variantEditSkuPayload(form, sku))
      onClose()
    } catch (err) {
      setError(apiErrorText(err) ?? t('variant.saveFailed'))
    } finally {
      setSaving(false)
    }
  }

  return (
    // Plain div, not <form> -- same nested-<form>-inside-the-page's-
    // outer-form avoidance as the Add Variant/Add SKU panels.
    <div className="mt-2 space-y-3 p-3 rounded-lg dark:bg-night-800/60 bg-gray-50 border border-line">
      <div>
        <label className={fieldLabelClass}>{t('variant.name')}</label>
        <input
          type="text" required placeholder={t('variant.name')} value={form.name} onChange={e => set('name', e.target.value)}
          className={fieldInputClass}
        />
      </div>
      {sku && (
        <>
          <div className="space-y-1.5">
            <p className="text-2xs font-semibold text-ink-3 caps-label-wide">{t('variant.pricing')}</p>
            <div className="grid grid-cols-2 gap-1.5">
              <div>
                <label className={fieldLabelClass}>{t('variant.skuCode')}</label>
                <input type="text" required placeholder={t('variant.skuCode')} value={form.skuCode} onChange={e => set('skuCode', e.target.value)} className={fieldInputClass} />
              </div>
              <div>
                <label className={fieldLabelClass}>{t('variant.currency')}</label>
                <select value={form.currency} onChange={e => set('currency', e.target.value)} className={fieldInputClass}>
                  {CURRENCIES.map(c => <option key={c} value={c}>{c}</option>)}
                </select>
              </div>
              <div>
                <label className={fieldLabelClass}>{t('variant.price')}</label>
                <input type="number" step="0.01" min="0" required placeholder="0.00" value={form.price} onChange={e => set('price', e.target.value)} className={fieldInputClass} />
              </div>
              <div>
                <label className={fieldLabelClass}>{t('variant.salePrice')}</label>
                <input type="number" step="0.01" min="0" placeholder={t('variant.optional')} value={form.salePrice} onChange={e => set('salePrice', e.target.value)} className={fieldInputClass} />
              </div>
              <div>
                <label className={fieldLabelClass}>{t('variant.saleEnds')}</label>
                <input type="date" title={t('variant.saleEndsOptional')} value={form.saleEndDate} onChange={e => set('saleEndDate', e.target.value)} className={fieldInputClass} />
              </div>
              <div>
                <label className={fieldLabelClass}>{t('variant.status')}</label>
                <select value={form.status} onChange={e => set('status', e.target.value as SKUStatus)} className={fieldInputClass}>
                  <option value="active">{t('sku.active')}</option>
                  <option value="inactive">{t('sku.inactive')}</option>
                </select>
              </div>
            </div>
          </div>
          <SKUCurrencyPricesEditor skuId={sku.id} />
          <div className="space-y-1.5">
            <p className="text-2xs font-semibold text-ink-3 caps-label-wide">{t('variant.shipping')}</p>
            <label className="flex items-center gap-1.5 text-2xs text-ink-3">
              <input type="checkbox" checked={form.isVirtual} onChange={e => set('isVirtual', e.target.checked)} />
              {t('variant.virtual')}
            </label>
            {!form.isVirtual && <DimensionInputsGrid values={form} onChange={set} />}
          </div>
        </>
      )}
      {error && <p className="text-2xs text-danger">{error}</p>}
      <div className="flex items-center gap-2 pt-1 border-t border-line">
        <button
          type="button" disabled={saving || !form.name || (!!sku && !form.skuCode)} onClick={save}
          className="px-2.5 py-1 rounded bg-accent-solid text-white text-2xs font-medium hover:bg-accent-solid-hover disabled:opacity-50 transition-colors"
        >
          {saving ? t('common:actions.saving') : t('common:actions.save')}
        </button>
        <button
          type="button" disabled={saving} onClick={onClose}
          className="px-2.5 py-1 rounded bg-fill text-2xs text-ink-3 hover:bg-hover transition-colors disabled:opacity-50"
        >
          {t('common:actions.cancel')}
        </button>
      </div>
    </div>
  )
}
