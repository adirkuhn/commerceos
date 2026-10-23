import { useEffect, useState } from 'react'
import { Plus, Search } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Money } from '../format/amountVisibility'
import { usePriceList } from './hooks/usePriceList'
import { applyDiscount, listPrice } from './pricing'
import { formatQuantity } from './format'
import type { PriceListRow } from './types'

const MIN_CHARS = 2

// Finds a product to add to the quote; results show the discounted price
// and stock so the right SKU is picked at a glance.
export function QuoteSearch({ token, discount, isAdded, onAdd }: {
  token: string
  discount: number
  isAdded: (skuId: string) => boolean
  onAdd: (row: PriceListRow) => void
}) {
  const { t } = useTranslation('priceList')
  const [input, setInput] = useState('')
  const [search, setSearch] = useState('')
  useEffect(() => {
    const timeout = setTimeout(() => setSearch(input.trim()), 300)
    return () => clearTimeout(timeout)
  }, [input])
  const enabled = search.length >= MIN_CHARS
  const { data, loading, error } = usePriceList(token, { search, perPage: 10 }, enabled)
  const rows = enabled ? data?.rows ?? [] : []

  return (
    <div className="relative max-w-xl">
      <Search size={13} className="absolute left-3 top-[15px] -translate-y-1/2 text-ink-4" />
      <input
        value={input}
        onChange={e => setInput(e.target.value)}
        placeholder={t('search')}
        aria-label={t('search')}
        className="w-full pl-8 pr-3 py-1.5 text-xs rounded-lg bg-raised border-line border text-ink placeholder-ink-4 outline-none focus:ring-1 focus:ring-focus"
      />
      {error && <p role="alert" className="mt-1 text-xs text-danger">{error}</p>}
      {enabled && !loading && rows.length === 0 && !error && <p className="mt-1 text-xs text-ink-4">{t('quote.noResults')}</p>}
      {rows.length > 0 && (
        <ul className="mt-1 rounded-lg border border-line bg-raised divide-y divide-line/60 max-h-72 overflow-auto">
          {rows.map(row => {
            const added = isAdded(row.sku_id)
            return (
              <li key={row.sku_id} className="flex items-center gap-3 px-3 py-2 text-xs">
                <div className="flex-1 min-w-0">
                  <p className="text-ink truncate">{row.product_name}{row.variant_name && <span className="text-ink-3"> — {row.variant_name}</span>}</p>
                  <p className="text-2xs font-mono text-ink-4">{row.sku_code} · {formatQuantity(row, row.available_quantity)}</p>
                </div>
                <Money minor={applyDiscount(listPrice(row), discount)} currency={row.currency_code} className="tabular-nums text-ink-2" />
                <button
                  onClick={() => onAdd(row)}
                  disabled={added}
                  aria-label={`${t('quote.add')} ${row.product_name} ${row.sku_code}`}
                  className="flex items-center gap-1 px-2 py-1 rounded-md bg-accent-solid text-white hover:bg-accent-solid-hover disabled:opacity-40 disabled:cursor-default"
                >
                  {added ? t('quote.added') : <><Plus size={12} /> {t('quote.add')}</>}
                </button>
              </li>
            )
          })}
        </ul>
      )}
    </div>
  )
}
