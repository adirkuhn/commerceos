import clsx from 'clsx'
import { useMemo } from 'react'
import { Download, Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import i18next from 'i18next'
import { Money } from '../format/amountVisibility'
import { downloadBlob } from '../newsletter/download'
import { usePriceList } from './hooks/usePriceList'
import { useQuoteLines } from './hooks/useQuoteLines'
import { applyDiscount, listPrice, quoteTotals, type PricedLine } from './pricing'
import { productLabel, quoteCsv } from './quoteCsv'
import { QuoteLineRow } from './QuoteLineRow'
import { QuoteSearch } from './QuoteSearch'
import type { PriceListRow } from './types'

const TH = 'px-3 py-2.5 font-medium text-ink-4 caps-label'

// A B2B quote: lines at the discounted price (each one can be typed over),
// totals and profit, exported as CSV with product, quantity and price only.
export function QuoteTab({ token, discount }: { token: string; discount: number }) {
  const { t } = useTranslation(['priceList', 'common'])
  const quote = useQuoteLines()
  const productIds = useMemo(() => [...new Set(quote.lines.map(l => l.productId))].sort(), [quote.lines])
  const { data, loading, error } = usePriceList(token, { productIds, perPage: 200 }, productIds.length > 0)
  const rowBySku = useMemo(() => new Map((data?.rows ?? []).map(r => [r.sku_id, r])), [data])
  const pricesIncludeTax = data?.pricesIncludeTax ?? true

  const priced: (PricedLine & { skuId: string })[] = quote.lines.flatMap(line => {
    const row = rowBySku.get(line.skuId)
    return row ? [{ skuId: line.skuId, row, quantity: line.quantity, unitPrice: line.unitPrice ?? applyDiscount(listPrice(row), discount) }] : []
  })
  const totals = quoteTotals(priced, pricesIncludeTax)
  const currency = priced[0]?.row.currency_code ?? ''

  function exportCsv() {
    const csv = quoteCsv(
      priced.map(l => ({ product: productLabel(l.row), quantity: l.quantity, unitPriceMinor: l.unitPrice })),
      [t('quote.csvProduct'), t('quote.csvQuantity'), t('quote.csvUnitPrice')],
      navigator.language || i18next.language,
    )
    const date = new Date().toISOString().slice(0, 10)
    downloadBlob(new Blob(['﻿', csv], { type: 'text/csv;charset=utf-8' }), t('quote.fileName', { date }))
  }

  return (
    <>
      <div className="flex flex-wrap items-start gap-3 px-5 py-3 border-b border-line bg-sunken shrink-0">
        <div className="flex-1 min-w-[260px]">
          <QuoteSearch token={token} discount={discount}
            isAdded={skuId => quote.lines.some(l => l.skuId === skuId)}
            onAdd={(row: PriceListRow) => quote.add(row.product_id, row.sku_id)} />
        </div>
        <div className="flex items-center gap-2">
          <button onClick={quote.clear} disabled={quote.lines.length === 0}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-line text-xs text-ink-2 hover:bg-hover disabled:opacity-40">
            <Trash2 size={13} /> {t('quote.clear')}
          </button>
          <button onClick={exportCsv} disabled={priced.length === 0}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover disabled:opacity-40">
            <Download size={13} /> {t('quote.export')}
          </button>
        </div>
      </div>
      <div className="flex-1 overflow-auto">
        {error && <p role="alert" className="px-5 py-2 text-xs text-danger">{error}</p>}
        <table className="w-full text-xs">
          <thead className="sticky top-0 z-10 bg-sunken border-b border-line">
            <tr>
              <th className={clsx(TH, 'text-left')}>{t('col.product')}</th>
              <th className={clsx(TH, 'text-left')}>{t('col.expiry')}</th>
              <th className={clsx(TH, 'text-right')}>{t('col.available')}</th>
              <th className={clsx(TH, 'text-right')}>{t('col.cost')}</th>
              <th className={clsx(TH, 'text-right')}>{t('col.unitPrice')}</th>
              <th className={clsx(TH, 'text-right')}>{t('col.quantity')}</th>
              <th className={clsx(TH, 'text-right')}>{t('col.lineTotal')}</th>
              <th className={clsx(TH, 'text-right')}>{t('col.profit')}</th>
              <th className={clsx(TH, 'text-right')}>{t('col.lineProfit')}</th>
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
                <QuoteLineRow key={line.skuId} line={line} row={row} pricesIncludeTax={pricesIncludeTax}
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
                  {t('profitNote')}{!totals.profitComplete && ` * ${t('quote.profitPartial')}`}
                </td>
              </tr>
            </tfoot>
          )}
        </table>
      </div>
    </>
  )
}
