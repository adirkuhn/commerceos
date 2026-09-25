import { useState } from 'react'
import { ArrowLeft, Plus, Trash2 } from 'lucide-react'
import clsx from 'clsx'
import { useAuth } from '../auth/AuthContext'
import type { SalesOrderStatus, SalesOrderDetail, CreateSalesOrderInput, CreateSalesOrderLineItemInput, Address } from './types'
import { useWarehouses } from '../inventory/hooks/useWarehouses'
import { ProductSkuPicker } from '../components/common/ProductSkuPicker'
import type { SKU } from '../pms/types'
import { useTaxClasses } from '../tax/hooks/useTaxClasses'
import { useTaxRates } from '../tax/hooks/useTaxRates'
import { useTenantSettings } from '../settings/hooks/useTenantSettings'
import { STATUS_OPTIONS } from './salesOrderDisplay'
import { formatMoney } from '../format/money'

// ── Create Sales Order Page ──────────────────────────────────────────────────

interface DraftLineItem {
  key: string
  skuId: string
  label: string
  quantity: string
  price: string
  // null means "use the tenant's default tax class" -- same convention as
  // pms.SKU.TaxClassID.
  taxClassId: string | null
}

function emptyAddress(): Address {
  return { first_name: '', last_name: '', company: '', address_1: '', address_2: '', city: '', state: '', postcode: '', country: '', email: '', phone: '' }
}

export function CreateSalesOrderPage({ onCreate, onClose }: { onCreate: (input: CreateSalesOrderInput) => Promise<SalesOrderDetail>; onClose: () => void }) {
  const { token } = useAuth()
  const { warehouses } = useWarehouses(token ?? '')
  const { classes: taxClasses } = useTaxClasses(token ?? '')
  const { rates: taxRates } = useTaxRates(token ?? '')
  const { settings: tenantSettings } = useTenantSettings(token ?? '')

  const [warehouseId, setWarehouseId] = useState('')
  const [currencyCode, setCurrencyCode] = useState('USD')
  const [status, setStatus] = useState<SalesOrderStatus>('pending')
  const [occurredAt, setOccurredAt] = useState(() => new Date().toISOString().slice(0, 16))
  const [customerNote, setCustomerNote] = useState('')
  const [billing, setBilling] = useState<Address>(emptyAddress())
  const [shipping, setShipping] = useState<Address>(emptyAddress())
  const [lineItems, setLineItems] = useState<DraftLineItem[]>([])
  const [pickerOpen, setPickerOpen] = useState(false)
  const [saving, setSaving] = useState(false)
  const [formError, setFormError] = useState('')

  function addSku(sku: SKU, label: string) {
    setLineItems(prev => [...prev, {
      key: `${sku.id}-${prev.length}`, skuId: sku.id, label,
      quantity: '1', price: (sku.price_minor_units / 100).toFixed(2),
      taxClassId: sku.tax_class_id ?? null,
    }])
    setPickerOpen(false)
  }

  function updateLineItem(key: string, field: 'quantity' | 'price', value: string) {
    setLineItems(prev => prev.map(li => (li.key === key ? { ...li, [field]: value } : li)))
  }

  function removeLineItem(key: string) {
    setLineItems(prev => prev.filter(li => li.key !== key))
  }

  const defaultTaxClassId = taxClasses.find(c => c.is_default)?.id ?? null

  // Per line: subtotal===total (no discount UI here), tax split out from it
  // using the tenant's prices_include_tax convention -- same math as the
  // product form's price preview, just applied per order line and summed.
  function lineTax(li: DraftLineItem): { minorUnits: number; taxMinorUnits: number } {
    const qty = parseInt(li.quantity, 10) || 0
    const price = Math.round(parseFloat(li.price || '0') * 100)
    const minorUnits = qty * price
    const taxClassId = li.taxClassId ?? defaultTaxClassId
    const rate = taxClassId && tenantSettings?.default_tax_country
      ? taxRates.find(r => r.tax_class_id === taxClassId && r.country === tenantSettings.default_tax_country)
      : undefined
    if (!rate) return { minorUnits, taxMinorUnits: 0 }
    const rateFraction = rate.rate_basis_points / 10000
    const taxMinorUnits = tenantSettings?.prices_include_tax
      ? minorUnits - Math.round(minorUnits / (1 + rateFraction))
      : Math.round(minorUnits * rateFraction)
    return { minorUnits, taxMinorUnits }
  }

  const totalMinorUnits = lineItems.reduce((sum, li) => {
    const { minorUnits, taxMinorUnits } = lineTax(li)
    // prices_include_tax: the line price already carries the tax, so the
    // order total is just the sum of line amounts. Exclusive prices: tax is
    // added on top of each line to reach the order total.
    return sum + minorUnits + (tenantSettings?.prices_include_tax ? 0 : taxMinorUnits)
  }, 0)
  const taxTotalMinorUnits = lineItems.reduce((sum, li) => sum + lineTax(li).taxMinorUnits, 0)

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault()
    setFormError('')
    if (lineItems.length === 0) {
      setFormError('Add at least one line item')
      return
    }
    setSaving(true)
    try {
      const input: CreateSalesOrderInput = {
        warehouse_id: warehouseId, currency_code: currencyCode, status,
        occurred_at: new Date(occurredAt).toISOString(),
        customer_note: customerNote, billing, shipping,
        total_minor_units: totalMinorUnits,
        tax_total_minor_units: taxTotalMinorUnits,
        line_items: lineItems.map((li): CreateSalesOrderLineItemInput => {
          const qty = parseInt(li.quantity, 10) || 0
          const price = Math.round(parseFloat(li.price || '0') * 100)
          const { minorUnits, taxMinorUnits } = lineTax(li)
          return {
            sku_id: li.skuId, quantity: qty, price_minor_units: price,
            subtotal_minor_units: minorUnits, subtotal_tax_minor_units: taxMinorUnits,
            total_minor_units: minorUnits, total_tax_minor_units: taxMinorUnits,
          }
        }),
      }
      await onCreate(input)
      onClose()
    } catch (err) {
      setFormError(err instanceof Error ? err.message : 'Failed to create sales order')
    } finally {
      setSaving(false)
    }
  }

  const labelClass = 'block text-2xs font-medium text-ink-4 uppercase tracking-wider mb-1.5'
  const inputClass = 'w-full px-3 py-2 text-xs rounded-lg bg-raised border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus transition-shadow'

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-6 py-4 border-b border-line bg-sunken shrink-0">
        <div>
          <button
            onClick={onClose}
            className="flex items-center gap-1 text-2xs font-medium text-ink-4 hover:text-ink-2 transition-colors mb-1"
          >
            <ArrowLeft size={11} /> Sales Order
          </button>
          <h1 className="text-xl font-semibold text-ink leading-tight">New sales order</h1>
        </div>
        <div className="flex items-center gap-2">
          <button
            type="button" onClick={onClose}
            className="px-4 py-2 rounded-lg border border-line bg-raised text-xs font-semibold text-ink-2 hover:bg-hover transition-colors"
          >
            Discard
          </button>
          <button
            type="submit" form="create-sales-order-form" disabled={saving || !warehouseId}
            className="px-4 py-2 rounded-lg bg-accent-solid text-white text-xs font-semibold hover:bg-accent-solid-hover disabled:opacity-50 transition-colors"
          >
            {saving ? 'Creating…' : 'Create sales order'}
          </button>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto">
        <div className="max-w-3xl mx-auto p-6 space-y-5">
          <form id="create-sales-order-form" onSubmit={handleCreate} className="space-y-5">
            <div className="rounded-xl border border-line p-4 space-y-3">
              <h3 className="text-xs font-semibold text-ink">General</h3>
              <div className="grid grid-cols-3 gap-3">
                <div>
                  <label htmlFor="so-warehouse" className={labelClass}>Warehouse</label>
                  <select id="so-warehouse" required value={warehouseId} onChange={e => setWarehouseId(e.target.value)} className={inputClass}>
                    <option value="">Select…</option>
                    {warehouses.map(w => <option key={w.id} value={w.id}>{w.name}</option>)}
                  </select>
                </div>
                <div>
                  <label htmlFor="so-status" className={labelClass}>Status</label>
                  <select id="so-status" value={status} onChange={e => setStatus(e.target.value as SalesOrderStatus)} className={inputClass}>
                    {STATUS_OPTIONS.map(s => <option key={s} value={s}>{s}</option>)}
                  </select>
                </div>
                <div>
                  <label htmlFor="so-currency" className={labelClass}>Currency</label>
                  <input id="so-currency" type="text" value={currencyCode} onChange={e => setCurrencyCode(e.target.value.toUpperCase())} className={inputClass} />
                </div>
                <div className="col-span-2">
                  <label htmlFor="so-occurred-at" className={labelClass}>Date</label>
                  <input id="so-occurred-at" type="datetime-local" value={occurredAt} onChange={e => setOccurredAt(e.target.value)} className={inputClass} />
                </div>
              </div>
              <div>
                <label htmlFor="so-note" className={labelClass}>Customer note</label>
                <input id="so-note" type="text" value={customerNote} onChange={e => setCustomerNote(e.target.value)} className={inputClass} />
              </div>
              {formError && (
                <p className="text-xs text-danger bg-red-50 dark:bg-red-500/10 rounded-lg px-2 py-1.5">{formError}</p>
              )}
            </div>

            <div className="rounded-xl border border-line p-4 space-y-3">
              <div className="flex items-center justify-between">
                <h3 className="text-xs font-semibold text-ink">Line items</h3>
                <button
                  type="button" onClick={() => setPickerOpen(v => !v)}
                  className="flex items-center gap-1 text-2xs font-medium text-accent hover:text-accent"
                >
                  <Plus size={12} /> Add product
                </button>
              </div>
              {pickerOpen && <ProductSkuPicker onSelect={addSku} onClose={() => setPickerOpen(false)} />}
              {lineItems.length === 0 && <p className="text-xs text-ink-4">No line items yet</p>}
              {lineItems.length > 0 && (
                <table className="w-full text-xs">
                  <thead>
                    <tr className="border-b border-line">
                      <th className="text-left py-1.5 font-medium text-ink-4">Product</th>
                      <th className="text-right py-1.5 font-medium text-ink-4 w-20">Qty</th>
                      <th className="text-right py-1.5 font-medium text-ink-4 w-24">Price</th>
                      <th className="w-8" />
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-line/60">
                    {lineItems.map(li => (
                      <tr key={li.key}>
                        <td className="py-1.5 text-ink">{li.label}</td>
                        <td className="py-1.5">
                          <input
                            type="number" min="1" value={li.quantity}
                            onChange={e => updateLineItem(li.key, 'quantity', e.target.value)}
                            className="w-16 px-2 py-1 text-xs text-right rounded bg-raised border border-line text-ink-2"
                          />
                        </td>
                        <td className="py-1.5">
                          <input
                            type="number" step="0.01" min="0" value={li.price}
                            onChange={e => updateLineItem(li.key, 'price', e.target.value)}
                            className="w-20 px-2 py-1 text-xs text-right rounded bg-raised border border-line text-ink-2"
                          />
                        </td>
                        <td className="py-1.5 text-right">
                          <button type="button" onClick={() => removeLineItem(li.key)} aria-label="Remove line item" title="Remove line item" className="p-1 -m-1 rounded hover:bg-hover">
                            <Trash2 size={12} className="text-ink-4 hover:text-danger" />
                          </button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
              {lineItems.length > 0 && taxTotalMinorUnits > 0 && (
                <div className="flex justify-between text-xs text-ink-3 pt-2 border-t border-line">
                  <span>Tax</span>
                  <span>{formatMoney(taxTotalMinorUnits, currencyCode)}</span>
                </div>
              )}
              {lineItems.length > 0 && (
                <div className={clsx('flex justify-between text-xs font-semibold text-ink', taxTotalMinorUnits > 0 ? 'pt-1' : 'pt-2 border-t border-line')}>
                  <span>Total</span>
                  <span>{formatMoney(totalMinorUnits, currencyCode)}</span>
                </div>
              )}
            </div>

            <CreateAddressFields title="Billing address" prefix="new-billing" value={billing} onChange={setBilling} showPhone />
            <CreateAddressFields title="Shipping address" prefix="new-shipping" value={shipping} onChange={setShipping} />
          </form>
        </div>
      </div>
    </div>
  )
}

function CreateAddressFields({ title, prefix, value, onChange, showPhone }: { title: string; prefix: string; value: Address; onChange: (a: Address) => void; showPhone?: boolean }) {
  const labelClass = 'block text-2xs font-medium text-ink-4 uppercase tracking-wider mb-1.5'
  const inputClass = 'w-full px-3 py-2 text-xs rounded-lg bg-raised border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus transition-shadow'
  const set = (field: keyof Address) => (e: React.ChangeEvent<HTMLInputElement>) => onChange({ ...value, [field]: e.target.value })
  const id = (field: string) => `${prefix}-${field}`
  const label = (text: string) => `${title.replace(' address', '')} ${text}`

  return (
    <div className="rounded-xl border border-line p-4 space-y-3">
      <h3 className="text-xs font-semibold text-ink">{title}</h3>
      <div className="grid grid-cols-2 gap-3">
        <div>
          <label htmlFor={id('first-name')} className={labelClass}>{label('first name')}</label>
          <input id={id('first-name')} type="text" value={value.first_name ?? ''} onChange={set('first_name')} className={inputClass} />
        </div>
        <div>
          <label htmlFor={id('last-name')} className={labelClass}>{label('last name')}</label>
          <input id={id('last-name')} type="text" value={value.last_name ?? ''} onChange={set('last_name')} className={inputClass} />
        </div>
        <div>
          <label htmlFor={id('city')} className={labelClass}>{label('city')}</label>
          <input id={id('city')} type="text" value={value.city ?? ''} onChange={set('city')} className={inputClass} />
        </div>
        <div>
          <label htmlFor={id('country')} className={labelClass}>{label('country')}</label>
          <input id={id('country')} type="text" value={value.country ?? ''} onChange={set('country')} className={inputClass} />
        </div>
        {showPhone && (
          <div>
            <label htmlFor={id('phone')} className={labelClass}>{label('phone')}</label>
            <input id={id('phone')} type="tel" value={value.phone ?? ''} onChange={set('phone')} className={inputClass} />
          </div>
        )}
      </div>
    </div>
  )
}
