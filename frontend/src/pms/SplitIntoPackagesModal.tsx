import { useEffect, useRef, useState, type KeyboardEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { X } from 'lucide-react'
import { Modal, inputClass, labelClass } from '../inventory/components/Modal'
import { useWarehouses } from '../inventory/hooks/useWarehouses'
import * as inventoryApi from '../inventory/api'
import * as pmsApi from './api'
import type { Batch } from '../inventory/types'
import type { PackedWeightLine } from './types'
import { useDateFormat } from '../format/DateFormatProvider'
import { useMoneyFormat } from '../format/amountVisibility'
import { addPackageLine, formatKg, packagePriceMinorUnits, summarizePackages, type PackageLine } from './weightPackages'
import { apiErrorText } from '../api/apiErrorText'

// Where the packages are drawn from: a real lot (batchId) or a warehouse's
// general-ledger stock (batchId '').
export interface PackageSourceScope {
  warehouseId: string
  batchId: string
}

const scopeKey = (s: PackageSourceScope) => `${s.warehouseId}|${s.batchId}`
const batchScope = (b: Batch): PackageSourceScope => ({ warehouseId: b.warehouse_id, batchId: b.is_general_ledger ? '' : b.id })

// "Split this lot into packages": weigh each package, enter "10 × 380g"
// lines, and each weight becomes (or restocks) its own sellable variant --
// see backend application/pms.PackWeightPortionsUseCase for the mechanics.
export function SplitIntoPackagesModal({
  token, bulkSkuId, pricePerKiloMinorUnits, currencyCode, initialScope, onPacked, onClose,
}: {
  token: string
  bulkSkuId: string
  pricePerKiloMinorUnits: number
  currencyCode: string
  initialScope?: PackageSourceScope
  onPacked: (lines: PackedWeightLine[]) => void
  onClose: () => void
}) {
  const { t } = useTranslation(['pms', 'common'])
  const formatMoney = useMoneyFormat()
  const { formatDate } = useDateFormat()
  const { warehouses } = useWarehouses(token)
  const [sources, setSources] = useState<Batch[] | null>(null)
  const [sourceKey, setSourceKey] = useState(initialScope ? scopeKey(initialScope) : '')
  const [lines, setLines] = useState<PackageLine[]>([])
  const [quantity, setQuantity] = useState('1')
  const [weight, setWeight] = useState('')
  const [skuCodePrefix, setSkuCodePrefix] = useState('')
  const [reason, setReason] = useState('')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [packed, setPacked] = useState<PackedWeightLine[] | null>(null)
  const quantityRef = useRef<HTMLInputElement>(null)

  // Keyed on the scope's string form, not the object -- a caller passing
  // a fresh literal each render must not refetch every time.
  const initialKey = initialScope ? scopeKey(initialScope) : ''
  useEffect(() => {
    let cancelled = false
    inventoryApi.listBatches(token, bulkSkuId)
      .then(batches => {
        if (cancelled) return
        const withStock = batches.filter(b => b.quantity > 0 || scopeKey(batchScope(b)) === initialKey)
        setSources(withStock)
        // Default to the first real lot (it carries an expiry), else
        // general-ledger stock.
        const preferred = withStock.find(b => !b.is_general_ledger) ?? withStock[0]
        setSourceKey(key => key || (preferred ? scopeKey(batchScope(preferred)) : ''))
      })
      .catch(e => { if (!cancelled) setError(apiErrorText(e) ?? t('split.loadFailed')) })
    return () => { cancelled = true }
  }, [token, bulkSkuId, initialKey, t])

  const source = sources?.find(b => scopeKey(batchScope(b)) === sourceKey) ?? null
  const summary = summarizePackages(lines)
  const overLot = source !== null && summary.totalGrams > source.quantity
  const warehouseName = (id: string) => warehouses.find(w => w.id === id)?.name ?? id
  const money = (minor: number) => formatMoney(minor, currencyCode)

  const sourceLabel = (b: Batch) => {
    const common = { warehouse: warehouseName(b.warehouse_id), left: formatKg(b.quantity) }
    if (b.is_general_ledger) return t('split.sourceGeneral', common)
    const lot = b.lot_number || b.id
    return b.expiry_date
      ? t('split.sourceLot', { ...common, lot, date: formatDate(b.expiry_date) })
      : t('split.sourceLotNoExpiry', { ...common, lot })
  }

  const addLine = () => {
    const next = addPackageLine(lines, { weightGrams: Number(weight), quantity: Number(quantity) })
    if (next === lines) return
    setLines(next)
    setWeight('')
    setQuantity('1')
    quantityRef.current?.focus()
  }
  const addOnEnter = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key !== 'Enter') return
    e.preventDefault()
    addLine()
  }

  async function submit() {
    if (!source || lines.length === 0 || overLot) return
    setSaving(true)
    setError('')
    try {
      const scope = batchScope(source)
      const result = await pmsApi.packWeightPortions(token, bulkSkuId, {
        warehouse_id: scope.warehouseId, bulk_batch_id: scope.batchId || null,
        lines: lines.map(l => ({ weight_grams: l.weightGrams, quantity: l.quantity })),
        sku_code_prefix: skuCodePrefix || undefined, reason: reason || undefined,
      })
      setPacked(result.lines)
      onPacked(result.lines)
    } catch (e) {
      setError(apiErrorText(e) ?? t('split.failed'))
    } finally {
      setSaving(false)
    }
  }

  if (packed) {
    return (
      <Modal title={t('split.doneTitle')} onClose={onClose}>
        <div className="space-y-3">
          <ul className="rounded-lg border border-line divide-y divide-line">
            {packed.map(l => (
              <li key={l.sku_id} className="flex items-center justify-between gap-2 px-2 py-1.5 text-2xs">
                <span className="text-ink-2">{t('split.packageName', { weight: l.weight_grams })} <span className="text-ink-4">{l.sku_code}</span></span>
                <span className={l.created ? 'text-accent' : 'text-ink-3'}>
                  {t(l.created ? 'split.doneNew' : 'split.doneRestocked', { quantity: l.quantity })}
                </span>
              </li>
            ))}
          </ul>
          <div className="flex justify-end gap-2">
            <button type="button" onClick={() => { setPacked(null); setLines([]) }} className="px-3 py-1.5 text-xs rounded-lg text-ink-2 hover:bg-hover">
              {t('split.addMore')}
            </button>
            <button type="button" onClick={onClose} className="px-3 py-1.5 text-xs rounded-lg bg-accent-solid text-white hover:bg-accent-solid-hover">
              {t('common:actions.close')}
            </button>
          </div>
        </div>
      </Modal>
    )
  }

  return (
    <Modal title={t('split.title')} onClose={onClose}>
      <div className="space-y-3">
        <p className="text-2xs text-ink-4">{t('split.intro')}</p>

        {sources !== null && sources.length === 0 ? (
          <p className="text-2xs text-ink-3">{t('split.noStock')}</p>
        ) : (
          <div>
            <label className={labelClass} htmlFor="split-source">{t('split.source')}</label>
            <select id="split-source" value={sourceKey} onChange={e => setSourceKey(e.target.value)} className={inputClass}>
              {(sources ?? []).map(b => (
                <option key={scopeKey(batchScope(b))} value={scopeKey(batchScope(b))}>{sourceLabel(b)}</option>
              ))}
            </select>
          </div>
        )}

        <div className="flex items-end gap-2">
          <div className="w-20">
            <label className={labelClass} htmlFor="split-quantity">{t('split.quantity')}</label>
            <input
              id="split-quantity" ref={quantityRef} type="number" inputMode="numeric" min={1} step={1}
              value={quantity} onChange={e => setQuantity(e.target.value)} onKeyDown={addOnEnter} className={inputClass}
            />
          </div>
          <span className="pb-1.5 text-xs text-ink-4" aria-hidden>×</span>
          <div className="flex-1">
            <label className={labelClass} htmlFor="split-weight">{t('split.weight')}</label>
            <input
              id="split-weight" type="number" inputMode="numeric" min={1} step={1} placeholder="380"
              value={weight} onChange={e => setWeight(e.target.value)} onKeyDown={addOnEnter} className={inputClass}
            />
          </div>
          <button type="button" onClick={addLine} className="px-3 py-1.5 text-xs rounded-lg border border-line text-ink-2 hover:bg-hover">
            {t('split.add')}
          </button>
        </div>

        {lines.length > 0 && (
          <ul className="rounded-lg border border-line divide-y divide-line max-h-48 overflow-y-auto">
            {lines.map(l => (
              <li key={l.weightGrams} className="flex items-center justify-between gap-2 px-2 py-1 text-2xs">
                <span className="text-ink-2">{t('split.line', { quantity: l.quantity, weight: l.weightGrams })}</span>
                <span className="flex items-center gap-2 text-ink-3">
                  {t('split.eachPrice', { price: money(packagePriceMinorUnits(pricePerKiloMinorUnits, l.weightGrams)) })}
                  <button
                    type="button" onClick={() => setLines(lines.filter(x => x.weightGrams !== l.weightGrams))}
                    aria-label={t('split.remove', { weight: l.weightGrams })} className="p-0.5 rounded text-ink-4 hover:text-danger"
                  >
                    <X size={12} />
                  </button>
                </span>
              </li>
            ))}
          </ul>
        )}

        {lines.length > 0 && source && (
          <p className="text-2xs text-ink-3">
            {t('split.summary', {
              weights: summary.weights, packages: summary.packages, used: formatKg(summary.totalGrams), left: formatKg(source.quantity),
            })}
          </p>
        )}
        {overLot && <p className="text-2xs text-danger">{t('split.overLot')}</p>}

        <details className="text-2xs">
          <summary className="cursor-pointer text-ink-3">{t('split.moreOptions')}</summary>
          <div className="mt-2 space-y-2">
            <div>
              <label className={labelClass} htmlFor="split-prefix">{t('split.prefix')}</label>
              <input
                id="split-prefix" value={skuCodePrefix} onChange={e => setSkuCodePrefix(e.target.value)}
                placeholder={t('split.prefixPlaceholder')} className={inputClass}
              />
            </div>
            <div>
              <label className={labelClass} htmlFor="split-reason">{t('split.reason')}</label>
              <input id="split-reason" value={reason} onChange={e => setReason(e.target.value)} className={inputClass} />
            </div>
          </div>
        </details>

        {error && <p className="text-2xs text-danger">{error}</p>}
        <div className="flex justify-end gap-2 pt-1">
          <button type="button" onClick={onClose} className="px-3 py-1.5 text-xs rounded-lg text-ink-2 hover:bg-hover">
            {t('common:actions.cancel')}
          </button>
          <button
            type="button" onClick={submit} disabled={saving || lines.length === 0 || !source || overLot}
            className="px-3 py-1.5 text-xs rounded-lg bg-accent-solid text-white hover:bg-accent-solid-hover disabled:opacity-50"
          >
            {saving ? t('split.creating') : t('split.create')}
          </button>
        </div>
      </div>
    </Modal>
  )
}
