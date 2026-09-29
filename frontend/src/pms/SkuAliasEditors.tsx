import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuth } from '../auth/AuthContext'
import * as pmsApi from './api'
import type { SKUCurrencyPrice, BarcodeAlias, ScaleCodeAlias } from './types'
import { CURRENCIES, formatPrice, normalizeScaleCodeInput } from './productFormat'

// SKUCurrencyPricesEditor manages this SKU's manual per-currency price
// overrides -- absent one, a sales channel running in that currency prices
// the SKU via the tenant's configured exchange rate instead (see
// Settings > Localization). Collapsed by default (shows a count once any
// exist) since most SKUs never need one.
export function SKUCurrencyPricesEditor({ skuId }: { skuId: string }) {
  const { token } = useAuth()
  const { t } = useTranslation(['pms', 'common'])
  const [open, setOpen] = useState(false)
  const [prices, setPrices] = useState<SKUCurrencyPrice[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [newCurrency, setNewCurrency] = useState(CURRENCIES[0])
  const [newPrice, setNewPrice] = useState('')
  const [newSalePrice, setNewSalePrice] = useState('')
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    if (!open || !token) return
    setLoading(true)
    setError('')
    pmsApi.listSKUCurrencyPrices(token, skuId)
      .then(setPrices)
      .catch(err => setError(err instanceof Error ? err.message : t('alias.currencyLoadFailed')))
      .finally(() => setLoading(false))
  }, [open, token, skuId])

  const label = 'block text-2xs font-medium text-ink-4 uppercase tracking-wide mb-0.5'
  const input = 'w-full px-2 py-1 text-2xs rounded bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus'

  const handleAdd = async () => {
    if (!token || !newPrice) return
    setSaving(true)
    setError('')
    try {
      const result = await pmsApi.upsertSKUCurrencyPrice(token, skuId, {
        currency_code: newCurrency,
        price_minor_units: Math.round(parseFloat(newPrice) * 100),
        sale_price_minor_units: newSalePrice ? Math.round(parseFloat(newSalePrice) * 100) : null,
      })
      setPrices(prev => (prev.some(p => p.currency_code === result.currency_code) ? prev.map(p => (p.currency_code === result.currency_code ? result : p)) : [...prev, result]))
      setNewPrice('')
      setNewSalePrice('')
    } catch (err) {
      setError(err instanceof Error ? err.message : t('alias.currencySaveFailed'))
    } finally {
      setSaving(false)
    }
  }

  const handleDelete = async (currencyCode: string) => {
    if (!token) return
    setError('')
    try {
      await pmsApi.deleteSKUCurrencyPrice(token, skuId, currencyCode)
      setPrices(prev => prev.filter(p => p.currency_code !== currencyCode))
    } catch (err) {
      setError(err instanceof Error ? err.message : t('alias.currencyDeleteFailed'))
    }
  }

  return (
    <div className="space-y-1.5">
      <button
        type="button"
        onClick={() => setOpen(o => !o)}
        className="text-2xs text-accent hover:text-accent/80 transition-colors"
      >
        {open ? t('alias.hide') : '+'} {t('alias.currencyTitle')}{prices.length > 0 ? ` (${prices.length})` : ''}
      </button>
      {open && (
        <div className="p-2 rounded-lg dark:bg-night-900/60 bg-gray-100 border border-line space-y-1.5">
          <p className="text-2xs text-ink-4">
            {t('alias.currencyHint')}
          </p>
          {loading && <p className="text-2xs text-ink-4">{t('common:status.loading')}</p>}
          {prices.map(p => (
            <div key={p.currency_code} className="flex items-center justify-between text-2xs text-ink-2">
              <span>
                {p.currency_code}: {formatPrice(p.price_minor_units, p.currency_code)}
                {p.sale_price_minor_units != null && t('alias.sale', { price: formatPrice(p.sale_price_minor_units, p.currency_code) })}
              </span>
              <button type="button" onClick={() => handleDelete(p.currency_code)} className="text-2xs text-ink-4 hover:text-danger transition-colors">
                {t('common:actions.delete')}
              </button>
            </div>
          ))}
          <div className="grid grid-cols-3 gap-1.5 pt-1">
            <div>
              <label htmlFor={`sku-currency-override-currency-${skuId}`} className={label}>{t('alias.currency')}</label>
              <select id={`sku-currency-override-currency-${skuId}`} value={newCurrency} onChange={e => setNewCurrency(e.target.value)} className={input}>
                {CURRENCIES.map(c => <option key={c} value={c}>{c}</option>)}
              </select>
            </div>
            <div>
              <label htmlFor={`sku-currency-override-price-${skuId}`} className={label}>{t('alias.price')}</label>
              <input id={`sku-currency-override-price-${skuId}`} type="number" step="0.01" min="0" placeholder="0.00" value={newPrice} onChange={e => setNewPrice(e.target.value)} className={input} />
            </div>
            <div>
              <label htmlFor={`sku-currency-override-sale-price-${skuId}`} className={label}>{t('alias.salePrice')}</label>
              <input id={`sku-currency-override-sale-price-${skuId}`} type="number" step="0.01" min="0" placeholder={t('alias.optional')} value={newSalePrice} onChange={e => setNewSalePrice(e.target.value)} className={input} />
            </div>
          </div>
          <button
            type="button"
            onClick={handleAdd}
            disabled={saving || !newPrice}
            className="w-full py-1 rounded bg-accent-solid text-white text-2xs font-medium hover:bg-accent-solid-hover disabled:opacity-50 transition-colors"
          >
            {saving ? t('common:actions.saving') : t('alias.addUpdate')}
          </button>
          {error && <p className="text-2xs text-danger">{error}</p>}
        </div>
      )}
    </div>
  )
}

export function BarcodeAliasesEditor({ skuId }: { skuId: string }) {
  const { token } = useAuth()
  const { t } = useTranslation(['pms', 'common'])
  const [open, setOpen] = useState(false)
  const [aliases, setAliases] = useState<BarcodeAlias[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [newCode, setNewCode] = useState('')
  const [newUnitsPerScan, setNewUnitsPerScan] = useState('1')
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    if (!open || !token) return
    setLoading(true)
    setError('')
    pmsApi.listBarcodeAliases(token, skuId)
      .then(setAliases)
      .catch(err => setError(err instanceof Error ? err.message : t('alias.barcodesLoadFailed')))
      .finally(() => setLoading(false))
  }, [open, token, skuId])

  const label = 'block text-2xs font-medium text-ink-4 uppercase tracking-wide mb-0.5'
  const input = 'w-full px-2 py-1 text-2xs rounded bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus'

  const handleAdd = async () => {
    if (!token || !newCode.trim()) return
    setSaving(true)
    setError('')
    try {
      const result = await pmsApi.createBarcodeAlias(token, skuId, {
        code: newCode.trim(),
        units_per_scan: parseInt(newUnitsPerScan, 10) || 1,
      })
      setAliases(prev => [...prev, result])
      setNewCode('')
      setNewUnitsPerScan('1')
    } catch (err) {
      setError(err instanceof Error ? err.message : t('alias.barcodeAddFailed'))
    } finally {
      setSaving(false)
    }
  }

  const handleDelete = async (aliasId: string) => {
    if (!token) return
    setError('')
    try {
      await pmsApi.deleteBarcodeAlias(token, aliasId)
      setAliases(prev => prev.filter(a => a.id !== aliasId))
    } catch (err) {
      setError(err instanceof Error ? err.message : t('alias.barcodeDeleteFailed'))
    }
  }

  return (
    <div className="space-y-1.5">
      <button type="button" onClick={() => setOpen(o => !o)} className="text-2xs text-accent hover:text-accent/80 transition-colors">
        {open ? t('alias.hide') : '+'} {t('alias.barcodesTitle')}{aliases.length > 0 ? ` (${aliases.length})` : ''}
      </button>
      {open && (
        <div className="p-2 rounded-lg dark:bg-night-900/60 bg-gray-100 border border-line space-y-1.5">
          <p className="text-2xs text-ink-4">
            {t('alias.barcodesHint')}
          </p>
          {loading && <p className="text-2xs text-ink-4">{t('common:status.loading')}</p>}
          {aliases.map(a => (
            <div key={a.id} className="flex items-center justify-between text-2xs text-ink-2">
              <span>{a.code}{a.units_per_scan > 1 ? t('alias.perScan', { n: a.units_per_scan }) : ''}</span>
              <button type="button" onClick={() => handleDelete(a.id)} className="text-2xs text-ink-4 hover:text-danger transition-colors">
                {t('common:actions.delete')}
              </button>
            </div>
          ))}
          <div className="grid grid-cols-3 gap-1.5 pt-1">
            <div className="col-span-2">
              <label htmlFor={`barcode-alias-code-${skuId}`} className={label}>{t('alias.barcode')}</label>
              <input id={`barcode-alias-code-${skuId}`} type="text" value={newCode} onChange={e => setNewCode(e.target.value)} className={input} />
            </div>
            <div>
              <label htmlFor={`barcode-alias-units-${skuId}`} className={label}>{t('alias.unitsPerScan')}</label>
              <input id={`barcode-alias-units-${skuId}`} type="number" step="1" min="1" value={newUnitsPerScan} onChange={e => setNewUnitsPerScan(e.target.value)} className={input} />
            </div>
          </div>
          <button
            type="button" onClick={handleAdd} disabled={saving || !newCode.trim()}
            className="w-full py-1 rounded bg-accent-solid text-white text-2xs font-medium hover:bg-accent-solid-hover disabled:opacity-50 transition-colors"
          >
            {saving ? t('common:actions.saving') : t('common:actions.add')}
          </button>
          {error && <p className="text-2xs text-danger">{error}</p>}
        </div>
      )}
    </div>
  )
}

export function ScaleCodeAliasesEditor({ skuId }: { skuId: string }) {
  const { token } = useAuth()
  const { t } = useTranslation(['pms', 'common'])
  const [open, setOpen] = useState(false)
  const [aliases, setAliases] = useState<ScaleCodeAlias[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [newCode, setNewCode] = useState('')
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    if (!open || !token) return
    setLoading(true)
    setError('')
    pmsApi.listScaleCodeAliases(token, skuId)
      .then(setAliases)
      .catch(err => setError(err instanceof Error ? err.message : t('alias.scaleLoadFailed')))
      .finally(() => setLoading(false))
  }, [open, token, skuId])

  const label = 'block text-2xs font-medium text-ink-4 uppercase tracking-wide mb-0.5'
  const input = 'w-full px-2 py-1 text-2xs rounded bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus'

  const handleAdd = async () => {
    if (!token || !newCode.trim()) return
    setSaving(true)
    setError('')
    try {
      const result = await pmsApi.createScaleCodeAlias(token, skuId, { code: newCode.trim() })
      setAliases(prev => [...prev, result])
      setNewCode('')
    } catch (err) {
      setError(err instanceof Error ? err.message : t('alias.scaleAddFailed'))
    } finally {
      setSaving(false)
    }
  }

  const handleDelete = async (aliasId: string) => {
    if (!token) return
    setError('')
    try {
      await pmsApi.deleteScaleCodeAlias(token, aliasId)
      setAliases(prev => prev.filter(a => a.id !== aliasId))
    } catch (err) {
      setError(err instanceof Error ? err.message : t('alias.scaleDeleteFailed'))
    }
  }

  return (
    <div className="space-y-1.5">
      <button type="button" onClick={() => setOpen(o => !o)} className="text-2xs text-accent hover:text-accent/80 transition-colors">
        {open ? t('alias.hide') : '+'} {t('alias.scaleTitle')}{aliases.length > 0 ? ` (${aliases.length})` : ''}
      </button>
      {open && (
        <div className="p-2 rounded-lg dark:bg-night-900/60 bg-gray-100 border border-line space-y-1.5">
          <p className="text-2xs text-ink-4">
            {t('alias.scaleHint')}
          </p>
          {loading && <p className="text-2xs text-ink-4">{t('common:status.loading')}</p>}
          {aliases.map(a => (
            <div key={a.id} className="flex items-center justify-between text-2xs text-ink-2">
              <span>{a.code}</span>
              <button type="button" onClick={() => handleDelete(a.id)} className="text-2xs text-ink-4 hover:text-danger transition-colors">
                {t('common:actions.delete')}
              </button>
            </div>
          ))}
          <div className="grid grid-cols-2 gap-1.5 pt-1">
            <div>
              <label htmlFor={`scale-code-alias-${skuId}`} className={label}>{t('alias.scaleCode')}</label>
              <input id={`scale-code-alias-${skuId}`} type="text" value={newCode} onChange={e => setNewCode(normalizeScaleCodeInput(e.target.value))} maxLength={7} inputMode="numeric" className={input} />
            </div>
          </div>
          <button
            type="button" onClick={handleAdd} disabled={saving || !newCode.trim()}
            className="w-full py-1 rounded bg-accent-solid text-white text-2xs font-medium hover:bg-accent-solid-hover disabled:opacity-50 transition-colors"
          >
            {saving ? t('common:actions.saving') : t('common:actions.add')}
          </button>
          {error && <p className="text-2xs text-danger">{error}</p>}
        </div>
      )}
    </div>
  )
}
