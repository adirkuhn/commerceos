import { useEffect, useState } from 'react'
import { AlertTriangle, X } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { apiErrorText } from '../api/apiErrorText'
import { Dialog } from '../components/common/Dialog'
import type { SalesOrder } from '../oms/types'
import { useShippingAddresses } from '../customers/hooks/useShippingAddresses'
import { oneLineAddress } from '../customers/shippingAddresses/addressLine'
import { checkQuoteStock, convertQuote } from './quotesApi'
import type { ConvertQuoteResult, QuoteStockCheckLine } from './types'

type Step = 'form' | 'repeat' | 'repeatAgain' | 'short' | 'busy'

export type QuoteOrder = Pick<SalesOrder, 'id' | 'number' | 'external_order_id'>

export function orderLabel(o: QuoteOrder): string {
  return o.number || `#${o.external_order_id.slice(0, 8)}`
}

// Quote -> sales order: pick the warehouse (and optionally the order's
// number, and which of the customer's shipping addresses), confirm twice when the quote already became an order, check the
// stock, and on a shortfall either go back to fix the quote or create the
// order anyway (it then lands in Stock issues).
export function ConvertQuoteDialog({ token, quoteId, customerId = null, warehouses, existingOrders, productName, onClose, onCreated }: {
  token: string
  quoteId: string
  // The saved quote's customer: with more than one shipping address, the
  // order's one is picked here (their default preselected).
  customerId?: string | null
  warehouses: { id: string; name: string }[]
  existingOrders: QuoteOrder[]
  productName: (skuId: string) => string
  onClose: () => void
  onCreated: (result: ConvertQuoteResult) => void
}) {
  const { t } = useTranslation(['quotes', 'common', 'customers'])
  const [warehouseId, setWarehouseId] = useState(warehouses[0]?.id ?? '')
  const [number, setNumber] = useState('')
  const [step, setStep] = useState<Step>('form')
  const [short, setShort] = useState<QuoteStockCheckLine[]>([])
  const [error, setError] = useState('')
  const { addresses } = useShippingAddresses(token, customerId)
  const [addressId, setAddressId] = useState('')
  const choosesAddress = addresses.length > 1
  const defaultAddressId = addresses.find(a => a.is_default)?.id ?? ''
  useEffect(() => { setAddressId(defaultAddressId) }, [defaultAddressId])

  async function create() {
    setStep('busy')
    setError('')
    try {
      onCreated(await convertQuote(token, quoteId, {
        warehouse_id: warehouseId, number: number.trim() || undefined,
        ...(choosesAddress && addressId ? { shipping_address_id: addressId } : {}),
      }))
    } catch (e) {
      setError(apiErrorText(e) ?? t('convert.failed'))
      setStep('form')
    }
  }

  async function checkThenCreate() {
    setStep('busy')
    setError('')
    try {
      const check = await checkQuoteStock(token, quoteId, warehouseId)
      if (check.has_shortfall) {
        setShort(check.lines.filter(l => l.short > 0))
        setStep('short')
        return
      }
    } catch (e) {
      setError(apiErrorText(e) ?? t('convert.failed'))
      setStep('form')
      return
    }
    await create()
  }

  const button = 'px-3 py-1.5 rounded-lg text-xs font-medium disabled:opacity-40'
  const primary = `${button} bg-accent-solid text-white hover:bg-accent-solid-hover`
  const secondary = `${button} border border-line text-ink-2 hover:bg-hover`

  return (
    <Dialog onClose={onClose} label={t('convert.title')} className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 dark:bg-black/60 bg-black/30" onClick={onClose} />
      <div className="relative w-full max-w-lg bg-raised rounded-xl shadow-2xl border border-line p-5">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-sm font-semibold text-ink">{t('convert.title')}</h3>
          <button onClick={onClose} aria-label={t('common:actions.close')}><X size={14} className="text-ink-4" /></button>
        </div>

        {(step === 'form' || step === 'busy') && (
          <div className="space-y-3">
            {existingOrders.length > 0 && (
              <p className="flex gap-2 p-2.5 rounded-lg bg-warning/10 text-xs text-ink-2">
                <AlertTriangle size={14} className="text-warning shrink-0" />
                {t('convert.alreadyOrdered', { orders: existingOrders.map(orderLabel).join(', ') })}
              </p>
            )}
            <label className="block text-2xs text-ink-4 caps-label">
              {t('convert.warehouse')}
              <select value={warehouseId} onChange={e => setWarehouseId(e.target.value)} aria-label={t('convert.warehouse')}
                className="mt-1.5 w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 normal-case">
                {warehouses.map(w => <option key={w.id} value={w.id}>{w.name}</option>)}
              </select>
            </label>
            {choosesAddress && (
              <label className="block text-2xs text-ink-4 caps-label">
                {t('customers:shippingAddresses.pick')}
                <select value={addressId} onChange={e => setAddressId(e.target.value)} aria-label={t('customers:shippingAddresses.pick')}
                  className="mt-1.5 w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 normal-case">
                  {addresses.map(a => <option key={a.id} value={a.id}>{[a.label, oneLineAddress(a)].filter(Boolean).join(' — ')}</option>)}
                </select>
              </label>
            )}
            <label className="block text-2xs text-ink-4 caps-label">
              {t('convert.number')}
              <input value={number} onChange={e => setNumber(e.target.value)} maxLength={64} aria-label={t('convert.number')}
                placeholder={t('convert.numberPlaceholder')}
                className="mt-1.5 w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 normal-case" />
            </label>
            <p className="text-2xs text-ink-4">{t('convert.hint')}</p>
            {error && <p role="alert" className="text-xs text-danger">{error}</p>}
            <div className="flex justify-end gap-2 pt-1">
              <button onClick={onClose} className={secondary}>{t('common:actions.cancel')}</button>
              <button onClick={() => (existingOrders.length > 0 ? setStep('repeat') : checkThenCreate())}
                disabled={!warehouseId || step === 'busy'} className={primary}>
                {step === 'busy' ? t('convert.working') : t('convert.submit')}
              </button>
            </div>
          </div>
        )}

        {step === 'repeat' && (
          <div className="space-y-3">
            <p className="text-xs text-ink-2">{t('convert.repeatQuestion', { orders: existingOrders.map(orderLabel).join(', ') })}</p>
            <div className="flex justify-end gap-2">
              <button onClick={() => setStep('form')} className={secondary}>{t('common:actions.cancel')}</button>
              <button onClick={() => setStep('repeatAgain')} className={primary}>{t('convert.repeatConfirm')}</button>
            </div>
          </div>
        )}

        {step === 'repeatAgain' && (
          <div className="space-y-3">
            <p className="text-xs text-ink-2">{t('convert.repeatAgainQuestion')}</p>
            <div className="flex justify-end gap-2">
              <button onClick={() => setStep('form')} className={secondary}>{t('common:actions.cancel')}</button>
              <button onClick={checkThenCreate} className={`${button} bg-danger text-white`}>{t('convert.repeatAgainConfirm')}</button>
            </div>
          </div>
        )}

        {step === 'short' && (
          <div className="space-y-3">
            <div role="alert" className="space-y-2">
              <p className="flex gap-2 text-xs text-ink-2">
                <AlertTriangle size={14} className="text-warning shrink-0" /> {t('convert.shortIntro')}
              </p>
              <table className="w-full text-xs">
                <thead>
                  <tr className="text-ink-4 border-b border-line">
                    <th className="text-left py-1 font-medium">{t('convert.colProduct')}</th>
                    <th className="text-right py-1 font-medium">{t('convert.colNeeded')}</th>
                    <th className="text-right py-1 font-medium">{t('convert.colAvailable')}</th>
                    <th className="text-right py-1 font-medium">{t('convert.colShort')}</th>
                  </tr>
                </thead>
                <tbody>
                  {short.map(l => (
                    <tr key={l.sku_id} className="border-b border-line/60">
                      <td className="py-1.5">{productName(l.sku_id)}</td>
                      <td className="py-1.5 text-right tabular-nums">{l.quantity}</td>
                      <td className="py-1.5 text-right tabular-nums">{l.available}</td>
                      <td className="py-1.5 text-right tabular-nums text-danger font-medium">{l.short}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
              <p className="text-2xs text-ink-4">{t('convert.shortHint')}</p>
            </div>
            <div className="flex justify-end gap-2">
              <button onClick={onClose} className={secondary}>{t('convert.backToFix')}</button>
              <button onClick={create} className={primary}>{t('convert.createAnyway')}</button>
            </div>
          </div>
        )}
      </div>
    </Dialog>
  )
}
