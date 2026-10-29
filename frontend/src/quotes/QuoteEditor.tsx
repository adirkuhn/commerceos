import clsx from 'clsx'
import { useEffect, useMemo, useState } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import { ArrowLeft, Download, FileText, ShoppingCart, Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import i18next from 'i18next'
import { apiErrorText } from '../api/apiErrorText'
import { Money } from '../format/amountVisibility'
import { downloadBlob } from '../newsletter/download'
import { useQueryString } from '../routing/useQueryState'
import { DiscountField } from '../pricelist/DiscountField'
import { CustomerPicker, type PickedCustomer } from '../customers/CustomerPicker'
import { useWarehouses } from '../inventory/hooks/useWarehouses'
import { ConvertQuoteDialog, orderLabel } from './ConvertQuoteDialog'
import { useQuoteOrders } from './hooks/useQuoteOrders'
import { usePriceList } from '../pricelist/hooks/usePriceList'
import { useQuoteLines } from './hooks/useQuoteLines'
import { useSavedQuote } from './hooks/useSavedQuotes'
import { applyDiscount, listPrice, parseDiscount, quoteTotals, type PricedLine } from '../pricelist/pricing'
import { productName, quoteCsv, type ExportLine } from './quoteCsv'
import { QuoteLineRow } from './QuoteLineRow'
import { openQuoteDocument } from './openQuoteDocument'
import { useDateFormat } from '../format/DateFormatProvider'
import { QuoteSaveBar } from './QuoteSaveBar'
import { MoneybirdDocumentCard } from '../moneybird/MoneybirdDocumentCard'
import { ProductPriceSearch } from '../pricelist/ProductPriceSearch'
import { createQuote, updateQuote } from './quotesApi'
import { buildQuoteInput, isSameQuote, savedQuoteInput, savedQuotePath } from './savedQuote'
import type { PriceListRow } from '../pricelist/types'

const TH = 'px-3 py-2.5 font-medium text-ink-4 caps-label'

// A B2B quote: lines at the discounted price (each one can be typed over),
// totals and profit, CSV/PDF export. /quotes/new is an unsaved quote,
// /quotes/<id> a saved one at the prices it was saved with; lines and the
// discount live in the URL (?lines=, ?discount=).
export function QuoteEditor({ token, quoteId, onClose }: { token: string; quoteId: string | null; onClose: () => void }) {
  const { t } = useTranslation(['quotes', 'priceList', 'common'])
  const navigate = useNavigate()
  const [discountRaw, setDiscount] = useQueryString('discount')
  const discount = parseDiscount(discountRaw)
  const quote = useQuoteLines()
  const productIds = useMemo(() => [...new Set(quote.lines.map(l => l.productId))].sort(), [quote.lines])
  const { data, loading, error, refresh } = usePriceList(token, { productIds, perPage: 200 }, productIds.length > 0)
  const rowBySku = useMemo(() => new Map((data?.rows ?? []).map(r => [r.sku_id, r])), [data])
  const pricesIncludeTax = data?.pricesIncludeTax ?? true

  const saved = useSavedQuote(token, quoteId)
  const [name, setName] = useState('')
  const [customer, setCustomer] = useState<PickedCustomer | null>(null)
  useEffect(() => {
    if (!saved.data) return
    setName(saved.data.name)
    setCustomer(saved.data.customer)
  }, [saved.data])
  // A bare /quotes/<id> link (the order page's "From quote") carries no
  // lines: open the quote as saved.
  const [searchParams] = useSearchParams()
  const bare = !searchParams.has('lines') && !searchParams.has('discount')
  useEffect(() => {
    if (saved.data && saved.data.id === quoteId && bare && saved.data.lines.length > 0) {
      navigate(savedQuotePath(saved.data), { replace: true })
    }
  }, [saved.data]) // eslint-disable-line react-hooks/exhaustive-deps
  const [saving, setSaving] = useState(false)
  const [pdfError, setPdfError] = useState('')
  const { formatDate } = useDateFormat()
  const [converting, setConverting] = useState(false)
  const { warehouses } = useWarehouses(token)
  const orders = useQuoteOrders(token, quoteId)
  const [saveError, setSaveError] = useState('')
  const input = buildQuoteInput(name, customer?.id ?? null, discount, quote.lines, rowBySku, saved.data?.currency_code)
  const dirty = quoteId
    ? saved.data !== null && !isSameQuote(input, savedQuoteInput(saved.data))
    : quote.lines.length > 0 || name.trim() !== '' || customer !== null

  async function persist(asNew: boolean) {
    setSaving(true)
    setSaveError('')
    try {
      const q = quoteId && !asNew ? await updateQuote(token, quoteId, input) : await createQuote(token, input)
      navigate(savedQuotePath(q), { replace: true })
      if (q.id === quoteId) saved.refresh()
    } catch (e) {
      setSaveError(apiErrorText(e) ?? t('quote.saveFailed'))
    } finally {
      setSaving(false)
    }
  }

  function startNew() {
    navigate('/quotes/new')
    setName('')
    setCustomer(null)
    setSaveError('')
  }

  const priced: (PricedLine & { skuId: string })[] = quote.lines.flatMap(line => {
    const row = rowBySku.get(line.skuId)
    return row ? [{ skuId: line.skuId, row, quantity: line.quantity, unitPrice: line.unitPrice ?? applyDiscount(listPrice(row), discount) }] : []
  })
  const totals = quoteTotals(priced, pricesIncludeTax)
  const currency = priced[0]?.row.currency_code ?? ''

  const exportLines: ExportLine[] = priced.map(l => ({
    sku: l.row.sku_code, product: productName(l.row), quantity: l.quantity, unitPriceMinor: l.unitPrice,
  }))

  function exportCsv() {
    const csv = quoteCsv(exportLines, {
      sku: t('print.sku'), product: t('print.product'), quantity: t('quote.csvQuantity'),
      unitPrice: t('print.unitPrice'), lineTotal: t('print.lineTotal'), total: t('print.orderTotal'),
    }, navigator.language || i18next.language)
    const date = new Date().toISOString().slice(0, 10)
    downloadBlob(new Blob(['\uFEFF', csv], { type: 'text/csv;charset=utf-8' }), t('quote.fileName', { date }))
  }

  async function downloadPdf() {
    setPdfError('')
    try {
      await openQuoteDocument(token, {
        reference: saved.data && saved.data.id === quoteId ? saved.data.reference : null,
        name, customerName: customer?.name ?? null, date: formatDate(new Date().toISOString()), currency, lines: exportLines,
      })
    } catch (e) {
      setPdfError(apiErrorText(e) ?? t('quote.saveFailed'))
    }
  }

  return (
    <>
      {converting && quoteId && (
        <ConvertQuoteDialog token={token} quoteId={quoteId} warehouses={warehouses} existingOrders={orders.data ?? []}
          productName={skuId => { const row = rowBySku.get(skuId); return row ? productName(row) : skuId }}
          onClose={() => setConverting(false)}
          onCreated={r => navigate(r.needs_stock_review ? '/stock-issues' : `/sales-orders/${r.sales_order_id}`)} />
      )}
      <QuoteSaveBar
        reference={saved.data && saved.data.id === quoteId ? saved.data.reference : null}
        name={name} onName={setName} dirty={dirty} saving={saving}
        customer={<CustomerPicker token={token} value={customer} onChange={setCustomer} />}
        error={saveError || (quoteId && saved.error) || ''}
        onSave={() => persist(false)} onSaveAsNew={() => persist(true)} onNew={startNew}
      />
      {(orders.data?.length ?? 0) > 0 && (
        <p data-testid="quote-orders" className="px-5 py-1.5 border-b border-line bg-surface text-2xs text-ink-3">
          {t('convert.orders')}{' '}
          {orders.data!.map((o, i) => (
            <span key={o.id}>{i > 0 && ', '}<Link to={`/sales-orders/${o.id}`} className="text-accent hover:underline">{orderLabel(o)}</Link></span>
          ))}
        </p>
      )}
      {quoteId && (
        <div className="px-5 py-2 border-b border-line">
          <MoneybirdDocumentCard sourceType="quote" sourceId={quoteId} blockedReason={dirty ? t('convert.saveFirst') : undefined} />
        </div>
      )}
      <div className="flex flex-wrap items-start gap-3 px-5 py-3 border-b border-line bg-sunken shrink-0">
        <button onClick={onClose} className="flex items-center gap-1 py-1.5 text-xs text-ink-3 hover:text-ink">
          <ArrowLeft size={13} /> {t('list.back')}
        </button>
        <div className="flex-1 min-w-[260px]">
          <ProductPriceSearch token={token} discount={discount}
            isAdded={skuId => quote.lines.some(l => l.skuId === skuId)}
            onAdd={(row: PriceListRow) => quote.add(row.product_id, row.sku_id)} />
        </div>
        <div className="flex items-center gap-2">
          <DiscountField value={discountRaw ?? ''} onChange={setDiscount} />
          <button onClick={quote.clear} disabled={quote.lines.length === 0}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-line text-xs text-ink-2 hover:bg-hover disabled:opacity-40">
            <Trash2 size={13} /> {t('quote.clear')}
          </button>
          <button onClick={exportCsv} disabled={priced.length === 0}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover disabled:opacity-40">
            <Download size={13} /> {t('quote.export')}
          </button>
          <button onClick={downloadPdf} disabled={priced.length === 0}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover disabled:opacity-40">
            <FileText size={13} /> {t('print.button')}
          </button>
          <button onClick={() => setConverting(true)} disabled={!quoteId || dirty || priced.length === 0}
            title={!quoteId || dirty ? t('convert.saveFirst') : undefined}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover disabled:opacity-40">
            <ShoppingCart size={13} /> {t('convert.button')}
          </button>
        </div>
      </div>
      <div className="flex-1 overflow-auto">
        {(error || pdfError) && <p role="alert" className="px-5 py-2 text-xs text-danger">{pdfError || error}</p>}
        <table className="w-full text-xs">
          <thead className="sticky top-0 z-10 bg-sunken border-b border-line">
            <tr>
              <th className={clsx(TH, 'text-left')}>{t('priceList:col.product')}</th>
              <th className={clsx(TH, 'text-left')}>{t('priceList:col.expiry')}</th>
              <th className={clsx(TH, 'text-right')}>{t('priceList:col.available')}</th>
              <th className={clsx(TH, 'text-right')}>{t('priceList:col.cost')}</th>
              <th className={clsx(TH, 'text-right')}>{t('priceList:col.unitPrice')}</th>
              <th className={clsx(TH, 'text-right')}>{t('priceList:col.quantity')}</th>
              <th className={clsx(TH, 'text-right')}>{t('priceList:col.lineTotal')}</th>
              <th className={clsx(TH, 'text-right')}>{t('priceList:col.profit')}</th>
              <th className={clsx(TH, 'text-right')}>{t('priceList:col.lineProfit')}</th>
              <th className="w-8" />
            </tr>
          </thead>
          <tbody className="divide-y divide-line/60">
            {quote.lines.length === 0 && (
              <tr><td colSpan={10} className="px-4 py-6 text-center text-ink-4">{t('quote.empty')}</td></tr>
            )}
            {quote.lines.map(line => {
              const row = rowBySku.get(line.skuId)
              if (!row) {
                return (
                  <tr key={line.skuId}>
                    <td colSpan={9} className="px-3 py-2 text-ink-4">{loading || !data ? t('quote.loadingLine') : t('quote.missingLine')}</td>
                    <td className="px-2 py-2">
                      <button onClick={() => quote.remove(line.skuId)} aria-label={t('quote.remove', { name: line.skuId })}
                        className="text-ink-4 hover:text-danger"><Trash2 size={12} /></button>
                    </td>
                  </tr>
                )
              }
              return (
                <QuoteLineRow key={line.skuId} token={token} line={line} onCostSaved={refresh} row={row} pricesIncludeTax={pricesIncludeTax}
                  discountedPrice={applyDiscount(listPrice(row), discount)}
                  onQuantity={q => quote.setQuantity(line.skuId, q)}
                  onUnitPrice={p => quote.setUnitPrice(line.skuId, p)}
                  onRemove={() => quote.remove(line.skuId)} />
              )
            })}
          </tbody>
          {priced.length > 0 && (
            <tfoot className="border-t-2 border-line bg-sunken">
              <tr>
                <td colSpan={6} className="px-3 py-2.5 text-right font-medium text-ink-3">{t('quote.total')}</td>
                <td data-testid="quote-total" className="px-3 py-2.5 text-right tabular-nums font-semibold text-ink"><Money minor={totals.total} currency={currency} /></td>
                <td className="px-3 py-2.5 text-right font-medium text-ink-3">{t('quote.totalProfit')}</td>
                <td data-testid="quote-profit" className="px-3 py-2.5 text-right tabular-nums font-semibold text-ink"
                  title={totals.profitComplete ? undefined : t('quote.profitPartial')}>
                  <Money minor={totals.profit} currency={currency} />{!totals.profitComplete && ' *'}
                </td>
                <td />
              </tr>
              <tr>
                <td colSpan={10} className="px-3 pb-2 text-right text-2xs text-ink-4">
                  {t('priceList:profitNote')}{!totals.profitComplete && ` * ${t('quote.profitPartial')}`}
                </td>
              </tr>
            </tfoot>
          )}
        </table>
      </div>
    </>
  )
}
