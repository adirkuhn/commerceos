import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Search, Trash2, ArrowLeft } from 'lucide-react'
import { getPromotion } from './api'
import { resolveBarcode } from '../pms/api'
import type { Promotion, PromotionInput, PromotionLine, PromotionStatus, PromotionType, CreatePromotionResult } from './types'
import { ProductSkuPicker } from '../components/common/ProductSkuPicker'
import { ClearanceBatchSelect } from './components/ClearanceBatchSelect'
import type { SKU } from '../pms/types'
import { PromotionPerformanceCard } from './PromotionPerformanceCard'
import { useDateFormat } from '../format/DateFormatProvider'

// ── Promotion Edit Page ──────────────────────────────────────────────────────

interface DraftLine extends PromotionLine {
  sku_code: string
  product_name: string
}

interface PromotionEditPageProps {
  token: string
  promotionId: string | null
  onCreate: (input: PromotionInput) => Promise<CreatePromotionResult>
  onUpdate: (id: string, input: PromotionInput) => Promise<Promotion>
  onClose: () => void
}

export function PromotionEditPage({ token, promotionId, onCreate, onUpdate, onClose }: PromotionEditPageProps) {
  const { formatDate } = useDateFormat()
  const { t } = useTranslation(['promotions', 'common'])
  const [name, setName] = useState('')
  const [type, setType] = useState<PromotionType>('percentage')
  const [status, setStatus] = useState<PromotionStatus>('draft')
  const [startsAt, setStartsAt] = useState('')
  const [endsAt, setEndsAt] = useState('')
  const [lines, setLines] = useState<DraftLine[]>([])
  const [skuCode, setSkuCode] = useState('')
  const [resolving, setResolving] = useState(false)
  const [pickerOpen, setPickerOpen] = useState(false)
  const [loadingDetail, setLoadingDetail] = useState(promotionId !== null)
  const [saving, setSaving] = useState(false)
  const [formError, setFormError] = useState('')

  useEffect(() => {
    if (!promotionId) return
    let cancelled = false
    setLoadingDetail(true)
    getPromotion(token, promotionId)
      .then(detail => {
        if (cancelled) return
        setName(detail.name)
        setType(detail.type)
        setStatus(detail.status)
        setStartsAt(detail.starts_at.slice(0, 16))
        setEndsAt(detail.ends_at ? detail.ends_at.slice(0, 16) : '')
        setLines((detail.lines ?? []).map(l => ({ ...l, product_name: '' })))
      })
      .catch(err => { if (!cancelled) setFormError(err instanceof Error ? err.message : t('edit.loadFailed')) })
      .finally(() => { if (!cancelled) setLoadingDetail(false) })
    return () => { cancelled = true }
  }, [token, promotionId])

  async function handleAddSKU() {
    const code = skuCode.trim()
    if (!code) return
    setFormError('')
    setResolving(true)
    try {
      const result = await resolveBarcode(token, code)
      if (addLine(result.sku.id, result.sku.sku_code, result.product_name)) setSkuCode('')
    } catch (err) {
      setFormError(err instanceof Error ? err.message : t('edit.resolveFailed'))
    } finally {
      setResolving(false)
    }
  }

  // addLine appends a draft line for skuId, returning false (with a form
  // error) when the SKU is already in this promotion.
  function addLine(skuId: string, skuCode: string, productName: string): boolean {
    if (lines.some(l => l.sku_id === skuId)) {
      setFormError(t('edit.duplicateSku'))
      return false
    }
    setLines(prev => [...prev, {
      id: `draft-${skuId}`, sku_id: skuId, sku_code: skuCode,
      product_name: productName, discount_type: type, discount_value: 0,
    }])
    return true
  }

  function handlePickSKU(sku: SKU, _label: string, productName: string) {
    setFormError('')
    if (addLine(sku.id, sku.sku_code, productName)) setPickerOpen(false)
  }

  function updateLineValue(skuId: string, value: string) {
    const parsed = value === '' ? 0 : parseInt(value, 10)
    setLines(prev => prev.map(l => (l.sku_id === skuId ? { ...l, discount_value: parsed } : l)))
  }

  function updateLineBatch(skuId: string, batchId: string | null) {
    setLines(prev => prev.map(l => (l.sku_id === skuId ? { ...l, batch_id: batchId } : l)))
  }

  function removeLine(skuId: string) {
    setLines(prev => prev.filter(l => l.sku_id !== skuId))
  }

  async function handleSave(e: React.FormEvent) {
    e.preventDefault()
    setFormError('')
    setSaving(true)
    try {
      const input: PromotionInput = {
        name, type, starts_at: new Date(startsAt).toISOString(),
        ends_at: endsAt ? new Date(endsAt).toISOString() : null,
        status,
        lines: lines.map(l => ({
          sku_id: l.sku_id, discount_type: type, discount_value: l.discount_value,
          ...(l.batch_id ? { batch_id: l.batch_id } : {}),
        })),
      }
      if (promotionId) {
        await onUpdate(promotionId, input)
      } else {
        const result = await onCreate(input)
        if (result.warnings.length > 0) {
          window.alert(result.warnings.join('\n'))
        }
      }
      onClose()
    } catch (err) {
      setFormError(err instanceof Error ? err.message : t('edit.saveFailed'))
    } finally {
      setSaving(false)
    }
  }

  const labelClass = 'block text-2xs font-medium text-ink-4 uppercase tracking-wider mb-1.5'
  const inputClass = 'w-full px-3 py-2 text-xs rounded-lg bg-raised border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus transition-shadow'

  if (loadingDetail) {
    return <div className="flex h-full items-center justify-center text-xs text-ink-4">{t('common:status.loading')}</div>
  }

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-6 py-4 border-b border-line bg-sunken shrink-0">
        <div>
          <button
            onClick={onClose}
            className="flex items-center gap-1 text-2xs font-medium text-ink-4 hover:text-ink-2 transition-colors mb-1"
          >
            <ArrowLeft size={11} /> {t('edit.back')}
          </button>
          <h1 className="text-xl font-semibold text-ink leading-tight">
            {promotionId ? name || t('edit.edit') : t('edit.new')}
          </h1>
        </div>
        <div className="flex items-center gap-2">
          <button
            type="button" onClick={onClose}
            className="px-4 py-2 rounded-lg border border-line bg-raised text-xs font-semibold text-ink-2 hover:bg-hover transition-colors"
          >
            {t('edit.discard')}
          </button>
          <button
            type="submit" form="promotion-form" disabled={saving || !name.trim() || lines.length === 0}
            className="px-4 py-2 rounded-lg bg-accent-solid text-white text-xs font-semibold hover:bg-accent-solid-hover disabled:opacity-50 transition-colors"
          >
            {saving ? t('common:actions.saving') : promotionId ? t('common:actions.save') : t('edit.create')}
          </button>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto">
        <div className="max-w-2xl mx-auto p-6 space-y-5">
          <form id="promotion-form" onSubmit={handleSave} className="space-y-5">
            <div className="rounded-xl border border-line p-4 space-y-3">
              <div className="flex items-center justify-between">
                <h3 className="text-xs font-semibold text-ink">{t('edit.general')}</h3>
                <select
                  aria-label={t('status.label')}
                  value={status}
                  onChange={e => setStatus(e.target.value as PromotionStatus)}
                  className="px-2 py-1 text-2xs rounded-lg bg-raised border border-line text-ink-2 outline-none focus-visible:ring-2 focus-visible:ring-focus/60"
                >
                  <option value="draft">{t('status.draft')}</option>
                  <option value="active">{t('status.active')}</option>
                  <option value="ended">{t('status.ended')}</option>
                </select>
              </div>
              <div>
                <label htmlFor="promotion-name" className={labelClass}>{t('edit.name')}</label>
                <input
                  id="promotion-name"
                  type="text" required autoFocus value={name} onChange={e => setName(e.target.value)}
                  className={inputClass}
                />
              </div>
              <div className="grid grid-cols-3 gap-3">
                <div>
                  <label htmlFor="promotion-type" className={labelClass}>{t('edit.type')}</label>
                  <select
                    id="promotion-type"
                    value={type} onChange={e => setType(e.target.value as PromotionType)}
                    className={inputClass}
                  >
                    <option value="percentage">{t('type.percentage')}</option>
                    <option value="fixed_amount">{t('type.fixed_amount')}</option>
                  </select>
                </div>
                <div>
                  <label htmlFor="promotion-starts" className={labelClass}>{t('edit.startsAt')}</label>
                  <input
                    id="promotion-starts"
                    type="datetime-local" required value={startsAt} onChange={e => setStartsAt(e.target.value)}
                    className={inputClass}
                  />
                </div>
                <div>
                  <label htmlFor="promotion-ends" className={labelClass}>{t('edit.endsAt')}</label>
                  <input
                    id="promotion-ends"
                    type="datetime-local" value={endsAt} onChange={e => setEndsAt(e.target.value)}
                    className={inputClass}
                  />
                </div>
              </div>
              {formError && (
                <p className="text-xs text-danger bg-red-50 dark:bg-red-500/10 rounded-lg px-2 py-1.5">{formError}</p>
              )}
            </div>

            <div className="rounded-xl border border-line p-4 space-y-3">
              <h3 className="text-xs font-semibold text-ink">{t('edit.skus')}</h3>
              <div className="flex items-center gap-2">
                <input
                  type="text" placeholder={t('edit.skuPlaceholder')} value={skuCode} onChange={e => setSkuCode(e.target.value)}
                  className={inputClass}
                />
                <button
                  type="button" onClick={handleAddSKU} disabled={resolving || !skuCode.trim()}
                  className="shrink-0 px-3 py-2 rounded-lg bg-raised border border-line text-xs font-medium text-ink-2 hover:bg-hover disabled:opacity-50 transition-colors"
                >
                  {resolving ? t('edit.resolving') : t('edit.addSku')}
                </button>
                <button
                  type="button" onClick={() => setPickerOpen(open => !open)}
                  className="shrink-0 inline-flex items-center gap-1 px-3 py-2 rounded-lg bg-raised border border-line text-xs font-medium text-ink-2 hover:bg-hover transition-colors"
                >
                  <Search size={12} /> {t('edit.searchProduct')}
                </button>
              </div>
              {pickerOpen && <ProductSkuPicker onSelect={handlePickSKU} onClose={() => setPickerOpen(false)} />}
              {lines.length === 0 && (
                <p className="text-xs text-ink-4">{t('edit.noSkus')}</p>
              )}
              <div className="divide-y divide-line/60">
                {lines.map(l => (
                  <div key={l.sku_id} className="flex flex-wrap sm:flex-nowrap items-center gap-3 py-2">
                    <div className="flex-1 min-w-0">
                      <p className="text-xs text-ink truncate">{l.product_name || l.sku_code}</p>
                      <p className="text-2xs font-mono text-ink-4">{l.sku_code}</p>
                      {l.finished_at && (
                        <p className="text-2xs text-ink-4">{t('edit.batchFinished', { date: formatDate(l.finished_at) })}</p>
                      )}
                    </div>
                    <div className="w-56">
                      <ClearanceBatchSelect token={token} skuId={l.sku_id} value={l.batch_id ?? null} onChange={id => updateLineBatch(l.sku_id, id)} />
                    </div>
                    <div className="w-28">
                      <label htmlFor={`discount-${l.sku_id}`} className="sr-only">{t('edit.discountValue')}</label>
                      <input
                        id={`discount-${l.sku_id}`}
                        type="number" min="0" value={l.discount_value === 0 ? '' : l.discount_value}
                        onChange={e => updateLineValue(l.sku_id, e.target.value)}
                        placeholder={type === 'percentage' ? t('edit.bps') : t('edit.minorUnits')}
                        className={inputClass}
                      />
                    </div>
                    <button aria-label={t('edit.remove')}
                      type="button" onClick={() => removeLine(l.sku_id)}
                      className="w-6 h-6 flex items-center justify-center rounded hover:bg-hover transition-colors shrink-0"
                      title={t('edit.remove')}
                    >
                      <Trash2 size={12} className="text-ink-3 hover:text-danger" />
                    </button>
                  </div>
                ))}
              </div>
            </div>
          </form>

          {promotionId && <PromotionPerformanceCard token={token} promotionId={promotionId} />}
        </div>
      </div>
    </div>
  )
}
