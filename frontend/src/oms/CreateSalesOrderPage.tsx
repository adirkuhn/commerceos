import { useEffect, useMemo, useState } from 'react'
import { ArrowLeft } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router-dom'
import { useAuth } from '../auth/AuthContext'
import { apiErrorText } from '../api/apiErrorText'
import type { ContactPerson, Customer } from '../customers/types'
import { useWarehouses } from '../inventory/hooks/useWarehouses'
import { resolveSkuTaxRate } from '../pms/skuTax'
import { useTenantSettings } from '../settings/hooks/useTenantSettings'
import { useTaxClasses } from '../tax/hooks/useTaxClasses'
import { useTaxRates } from '../tax/hooks/useTaxRates'
import { AddressFields } from './create/AddressFields'
import { OrderCustomerCard } from './create/OrderCustomerCard'
import { OrderItemsCard } from './create/OrderItemsCard'
import { OrderSummaryCard } from './create/OrderSummaryCard'
import { taxSetupGap } from '../tax/taxSetupGap'
import { MissingVatWarning } from '../tax/MissingVatWarning'
import { toEditInput } from './create/editDraft'
import { useEditDraft } from './hooks/useEditDraft'
import { draftTotals, pricesIncludeTax, toCreateInput, type OrderDraft } from './create/orderDraft'
import { STATUS_OPTIONS } from './salesOrderDisplay'
import { salesOrderStatusLabel } from './statusLabel'
import type { Address, CreateSalesOrderInput, EditSalesOrderInput, SalesOrderDetail } from './types'

const nowLocal = () => {
  const d = new Date()
  d.setMinutes(d.getMinutes() - d.getTimezoneOffset())
  return d.toISOString().slice(0, 16)
}

const emptyDraft = (): OrderDraft => ({
  warehouseId: '', status: 'processing', occurredAt: nowLocal(), number: '', note: '', customerId: null,
  billing: {}, shipping: {}, shippingSameAsBilling: true, lines: [], shipping_cost: 0, feeLabel: '', fee: 0,
})

const hasAny = (a?: Address) => !!a && Object.values(a).some(v => typeof v === 'string' && v.trim() !== '')

// New sales order: customer (pick or create), items at their current
// (sale) price, shipping and an extra fee, order details and addresses.
// The currency is the shop's; totals use the shop's VAT setting.
//
// With edit, the same form opens on an existing manual order (see
// canEditOrder) and saves it in place; its status is changed on the
// order page, not here.
export function CreateSalesOrderPage({ onCreate, onClose, edit }: {
  onCreate?: (input: CreateSalesOrderInput) => Promise<SalesOrderDetail>
  onClose: () => void
  edit?: { order: SalesOrderDetail; onSave: (input: EditSalesOrderInput) => Promise<SalesOrderDetail> }
}) {
  const { token } = useAuth()
  const { t } = useTranslation(['orders', 'common'])
  const { warehouses } = useWarehouses(token ?? '')
  const { classes, loading: classesLoading } = useTaxClasses(token ?? '')
  const { rates, loading: ratesLoading } = useTaxRates(token ?? '')
  const { settings, loading: settingsLoading } = useTenantSettings(token ?? '')
  const [draft, setDraft] = useState<OrderDraft>(emptyDraft)
  const [customer, setCustomer] = useState<Customer | null>(null)
  const [saving, setSaving] = useState(false)
  const [formError, setFormError] = useState('')
  const set = <K extends keyof OrderDraft>(key: K, value: OrderDraft[K]) => setDraft(d => ({ ...d, [key]: value }))
  const editing = useEditDraft(token ?? '', edit?.order)
  useEffect(() => {
    if (!editing.draft) return
    setDraft(editing.draft)
    setCustomer(editing.customer)
  }, [editing.draft]) // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    if (warehouses.length === 1 && !draft.warehouseId) set('warehouseId', warehouses[0].id)
  }, [warehouses]) // eslint-disable-line react-hooks/exhaustive-deps

  const currency = settings?.default_currency || 'EUR'
  const inclusive = pricesIncludeTax(settings)
  const defaultRate = resolveSkuTaxRate(rates, classes, null, settings?.default_tax_country)?.rate_basis_points ?? 0
  const taxGap = classesLoading || ratesLoading || settingsLoading ? null : taxSetupGap(classes, rates, settings?.default_tax_country)
  const totals = useMemo(() => draftTotals({
    lines: draft.lines, shipping: draft.shipping_cost, fee: draft.fee, defaultRateBasisPoints: defaultRate, pricesIncludeTax: inclusive,
  }), [draft.lines, draft.shipping_cost, draft.fee, defaultRate, inclusive])

  function chooseCustomer(c: Customer | null) {
    setCustomer(c)
    if (!c) {
      set('customerId', null)
      return
    }
    const billing: Address = { ...(c.billing ?? {}), email: c.billing?.email || c.email || '' }
    if (!billing.first_name && !billing.last_name) Object.assign(billing, { first_name: c.first_name, last_name: c.last_name })
    const shipping = hasAny(c.shipping) ? c.shipping! : {}
    setDraft(d => ({
      ...d, customerId: c.id, billing, shipping,
      shippingSameAsBilling: !hasAny(shipping) || JSON.stringify(shipping) === JSON.stringify(c.billing),
    }))
  }

  // A company order is addressed to its contact person: the billing name
  // (printed as "Attn." under the company) and, when known, their email.
  function chooseContactPerson(p: ContactPerson | null) {
    if (!p) return
    setDraft(d => ({
      ...d, billing: { ...d.billing, first_name: p.first_name, last_name: p.last_name, email: p.email || d.billing.email },
    }))
  }

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    if (!draft.warehouseId) return setFormError(t('form.needWarehouse'))
    if (draft.lines.length === 0) return setFormError(t('form.needItems'))
    setSaving(true)
    setFormError('')
    const ctx = { currency, defaultRateBasisPoints: defaultRate, pricesIncludeTax: inclusive }
    try {
      if (edit) await edit.onSave(toEditInput(draft, ctx))
      else await onCreate?.(toCreateInput(draft, ctx))
      onClose()
    } catch (err) {
      setFormError(apiErrorText(err) ?? t(edit ? 'edit.failed' : 'create.failed'))
    } finally {
      setSaving(false)
    }
  }

  const label = 'block text-2xs font-medium text-ink-4 mb-1'
  const input = 'w-full px-3 py-1.5 text-xs rounded-lg bg-raised border border-line text-ink outline-none focus:ring-1 focus:ring-focus'

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-6 py-4 border-b border-line bg-sunken shrink-0">
        <div>
          <button type="button" onClick={onClose} className="flex items-center gap-1 text-2xs font-medium text-ink-4 hover:text-ink-2 mb-1">
            <ArrowLeft size={11} /> {t('detail.back')}
          </button>
          <h1 className="text-xl font-semibold text-ink leading-tight">
            {edit ? t('edit.title', { number: edit.order.sales_order.number || edit.order.sales_order.external_order_id }) : t('create.title')}
          </h1>
        </div>
        <div className="flex items-center gap-2">
          <button type="button" onClick={onClose} className="px-4 py-2 rounded-lg border border-line bg-raised text-xs font-semibold text-ink-2 hover:bg-hover">
            {t('create.discard')}
          </button>
          <button type="submit" form="create-sales-order-form" disabled={saving || (!!edit && !editing.draft)}
            className="px-4 py-2 rounded-lg bg-accent-solid text-white text-xs font-semibold hover:bg-accent-solid-hover disabled:opacity-50">
            {edit ? (saving ? t('edit.saving') : t('edit.submit')) : (saving ? t('create.creating') : t('create.submit'))}
          </button>
        </div>
      </div>

      {edit && !editing.draft ? (
        <p role={editing.error ? 'alert' : 'status'} className={editing.error ? 'm-6 text-xs text-danger' : 'm-6 text-xs text-ink-4'}>
          {editing.error ? (apiErrorText(editing.error) ?? t('edit.loadFailed')) : t('edit.loading')}
        </p>
      ) : (
      <div className="flex-1 overflow-y-auto">
        <form id="create-sales-order-form" onSubmit={submit} className="max-w-6xl mx-auto p-6 grid gap-5 lg:grid-cols-3 items-start">
          <div className="lg:col-span-2 space-y-5">
            {formError && <p role="alert" className="text-xs text-danger bg-danger/10 rounded-lg px-3 py-2">{formError}</p>}
            {taxGap && (
              <p role="status" className="text-xs text-ink-2 bg-warning/10 rounded-lg px-3 py-2">
                {t(`form.taxGap.${taxGap}`, { country: settings?.default_tax_country })}{' '}
                <Link to="/settings/tax" className="underline">{t('form.taxGap.fix')}</Link>
              </p>
            )}
            {!taxGap && <MissingVatWarning products={draft.lines.filter(l => l.vatMissing).map(l => l.label)} />}
            <OrderCustomerCard token={token ?? ''} customer={customer} onChange={chooseCustomer} onContactPerson={chooseContactPerson}
              current={edit ? edit.order.sales_order.billing : undefined} />
            <OrderItemsCard token={token ?? ''} lines={draft.lines} lineTotals={totals.lineTotals} currency={currency} onChange={l => set('lines', l)} />
            <section className="rounded-xl border border-line bg-surface p-4 space-y-4">
              <h3 className="text-xs font-semibold text-ink">{t('form.addresses')}</h3>
              <div className="grid gap-5 md:grid-cols-2">
                <div className="space-y-2">
                  <h4 className="text-2xs font-semibold caps-label text-ink-4">{t('form.billing')}</h4>
                  <AddressFields prefix="billing" value={draft.billing} onChange={a => set('billing', a)} />
                </div>
                <div className="space-y-2">
                  <div className="flex items-center justify-between">
                    <h4 className="text-2xs font-semibold caps-label text-ink-4">{t('form.shippingAddress')}</h4>
                    <label className="flex items-center gap-1.5 text-2xs text-ink-3">
                      <input type="checkbox" checked={draft.shippingSameAsBilling} onChange={e => set('shippingSameAsBilling', e.target.checked)} />
                      {t('form.sameAsBilling')}
                    </label>
                  </div>
                  {!draft.shippingSameAsBilling && <AddressFields prefix="shipping" value={draft.shipping} onChange={a => set('shipping', a)} />}
                </div>
              </div>
            </section>
          </div>

          <div className="space-y-5 lg:sticky lg:top-0">
            <section className="rounded-xl border border-line bg-surface p-4 space-y-3">
              <h3 className="text-xs font-semibold text-ink">{t('form.order')}</h3>
              <div>
                <label htmlFor="so-warehouse" className={label}>{t('form.warehouse')}</label>
                <select id="so-warehouse" value={draft.warehouseId} onChange={e => set('warehouseId', e.target.value)} className={input}>
                  <option value="">{t('create.select')}</option>
                  {warehouses.map(w => <option key={w.id} value={w.id}>{w.name}</option>)}
                </select>
              </div>
              {!edit && (
                <div>
                  <label htmlFor="so-status" className={label}>{t('form.status')}</label>
                  <select id="so-status" value={draft.status} onChange={e => set('status', e.target.value as OrderDraft['status'])} className={input}>
                    {STATUS_OPTIONS.map(s => <option key={s} value={s}>{salesOrderStatusLabel(s)}</option>)}
                  </select>
                </div>
              )}
              <div>
                <label htmlFor="so-date" className={label}>{t('form.date')}</label>
                <input id="so-date" type="datetime-local" value={draft.occurredAt} onChange={e => set('occurredAt', e.target.value)} className={input} />
              </div>
              <div>
                <label htmlFor="so-number" className={label}>{t('form.number')}</label>
                <input id="so-number" value={draft.number} onChange={e => set('number', e.target.value)} placeholder={t('form.numberPlaceholder')} maxLength={64} className={input} />
              </div>
              <div>
                <label htmlFor="so-note" className={label}>{t('form.note')}</label>
                <textarea id="so-note" rows={2} value={draft.note} onChange={e => set('note', e.target.value)} className={input} />
              </div>
            </section>
            <OrderSummaryCard totals={totals} currency={currency} pricesIncludeTax={inclusive}
              shipping={draft.shipping_cost} feeLabel={draft.feeLabel} fee={draft.fee}
              onShipping={v => set('shipping_cost', v)} onFeeLabel={v => set('feeLabel', v)} onFee={v => set('fee', v)} />
          </div>
        </form>
      </div>
      )}
    </div>
  )
}
