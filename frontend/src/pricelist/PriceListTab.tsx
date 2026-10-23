import clsx from 'clsx'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import Pagination from '../components/common/Pagination'
import { Money } from '../format/amountVisibility'
import { useQueryPage, useQueryString } from '../routing/useQueryState'
import { usePriceList } from './hooks/usePriceList'
import { SearchField } from './SearchField'
import { applyDiscount, listPrice, unitEconomics } from './pricing'
import { belowTargetText, formatPercent, formatQuantity } from './format'
import type { PriceListRow } from './types'

const PAGE_SIZE = 50

const TH = 'px-3 py-2.5 font-medium text-ink-4 caps-label'

// View-only: every active SKU with its price and sale price before and
// after the discount, and the profit left at the discounted price.
export function PriceListTab({ token, discount }: { token: string; discount: number }) {
  const { t } = useTranslation(['priceList', 'common'])
  const [urlSearch, setUrlSearch] = useQueryString('q', { resets: ['page'] })
  const search = urlSearch ?? ''
  const [page, setPage] = useQueryPage()
  const [searchInput, setSearchInput] = useState(search)
  useEffect(() => {
    if (searchInput === search) return
    const timeout = setTimeout(() => setUrlSearch(searchInput), 300)
    return () => clearTimeout(timeout)
  }, [searchInput, search, setUrlSearch])
  useEffect(() => { setSearchInput(search) }, [search])

  const { data, loading, error } = usePriceList(token, { page, perPage: PAGE_SIZE, search })
  const rows = data?.rows ?? []

  return (
    <>
      <div className="flex flex-wrap items-center gap-3 px-5 py-3 border-b border-line bg-sunken shrink-0">
        <div className="flex-1 max-w-sm">
          <SearchField label={t('search')} value={searchInput} onChange={setSearchInput} />
        </div>
        <span className="text-2xs text-ink-4">{t('profitNote')}</span>
        {error && <span role="alert" className="text-xs text-danger ml-auto">{error}</span>}
      </div>
      <div className="flex-1 overflow-auto">
        <table className="w-full text-xs">
          <thead className="sticky top-0 z-10 bg-sunken border-b border-line">
            <tr>
              <th className={clsx(TH, 'text-left')}>{t('col.product')}</th>
              <th className={clsx(TH, 'text-right')}>{t('col.available')}</th>
              <th className={clsx(TH, 'text-right')}>{t('col.cost')}</th>
              <th className={clsx(TH, 'text-right')}>{t('col.price')}</th>
              <th className={clsx(TH, 'text-right')}>{t('col.discounted', { discount })}</th>
              <th className={clsx(TH, 'text-right')}>{t('col.salePrice')}</th>
              <th className={clsx(TH, 'text-right')}>{t('col.discounted', { discount })}</th>
              <th className={clsx(TH, 'text-right')}>{t('col.profit')}</th>
              <th className={clsx(TH, 'text-right')}>{t('col.markup')}</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-line/60">
            {loading && rows.length === 0 && (
              <tr><td colSpan={9} className="px-4 py-6 text-center text-ink-4">{t('common:status.loading')}</td></tr>
            )}
            {!loading && rows.length === 0 && (
              <tr><td colSpan={9} className="px-4 py-6 text-center text-ink-4">{t('empty')}</td></tr>
            )}
            {rows.map(row => (
              <PriceListTableRow key={row.sku_id} row={row} discount={discount} pricesIncludeTax={data?.pricesIncludeTax ?? true} />
            ))}
          </tbody>
        </table>
      </div>
      <Pagination page={page} pageSize={PAGE_SIZE} totalItems={data?.total ?? 0} onPageChange={setPage} />
    </>
  )
}

function PriceListTableRow({ row, discount, pricesIncludeTax }: { row: PriceListRow; discount: number; pricesIncludeTax: boolean }) {
  const { t } = useTranslation('priceList')
  const currency = row.currency_code
  const sale = row.sale_price_minor_units
  const economics = unitEconomics(row, applyDiscount(listPrice(row), discount), pricesIncludeTax)

  return (
    <tr data-testid={`price-row-${row.sku_id}`} className={clsx(economics.belowTarget && 'bg-danger/10')}>
      <td className="px-3 py-2 max-w-[260px]">
        <p className="font-medium text-ink truncate">{row.product_name}{row.variant_name && <span className="text-ink-3"> — {row.variant_name}</span>}</p>
        <p className="text-2xs font-mono text-ink-4 truncate">{row.sku_code}</p>
      </td>
      <td className="px-3 py-2 text-right tabular-nums text-ink-2">{formatQuantity(row, row.available_quantity)}</td>
      <td className="px-3 py-2 text-right tabular-nums text-ink-3">
        {row.cost_price_minor_units != null ? <Money minor={row.cost_price_minor_units} currency={currency} /> : '—'}
      </td>
      <td className="px-3 py-2 text-right tabular-nums text-ink-2"><Money minor={row.price_minor_units} currency={currency} /></td>
      <td className="px-3 py-2 text-right tabular-nums font-medium text-ink"><Money minor={applyDiscount(row.price_minor_units, discount)} currency={currency} /></td>
      <td className="px-3 py-2 text-right tabular-nums text-ink-2">{sale != null ? <Money minor={sale} currency={currency} /> : '—'}</td>
      <td className="px-3 py-2 text-right tabular-nums font-medium text-ink">
        {sale != null ? <Money minor={applyDiscount(sale, discount)} currency={currency} /> : '—'}
      </td>
      <td className={clsx('px-3 py-2 text-right tabular-nums', economics.belowTarget ? 'text-danger font-medium' : 'text-ink-2')}
        title={economics.belowTarget ? belowTargetText(row, economics.profit) : undefined}>
        {economics.profit != null ? <Money minor={economics.profit} currency={currency} /> : <span title={t('noCost')}>—</span>}
      </td>
      <td className={clsx('px-3 py-2 text-right tabular-nums', economics.belowTarget ? 'text-danger font-medium' : 'text-ink-3')}>
        {economics.markupBasisPoints != null ? formatPercent(economics.markupBasisPoints) : '—'}
      </td>
    </tr>
  )
}
