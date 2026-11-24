import { useState } from 'react'
import { Search, X } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useAuth } from '../../auth/AuthContext'
import { useProducts } from '../../pms/hooks/useProducts'
import * as pmsApi from '../../pms/api'
import type { SKU } from '../../pms/types'

// ProductSkuPicker finds a SKU by searching its product's name, then
// expanding the product to pick one of its SKUs -- shared by every form
// that adds SKU lines (purchase orders, sales orders, unmatched-SKU
// linking, promotions).
export interface ProductSkuPickerProps {
  // productName is passed separately so callers that show product and SKU
  // code on separate lines don't have to split label back apart.
  onSelect: (sku: SKU, label: string, productName: string) => void
  onClose: () => void
}

export function ProductSkuPicker({ onSelect, onClose }: ProductSkuPickerProps) {
  const { token } = useAuth()
  const { t } = useTranslation(['ui', 'common'])
  const [query, setQuery] = useState('')
  const { products } = useProducts(token ?? '', { search: query, perPage: 5 })
  const [expandedProductId, setExpandedProductId] = useState<string | null>(null)
  const [skus, setSkus] = useState<SKU[]>([])
  const [loadingSkus, setLoadingSkus] = useState(false)

  async function toggleProduct(productId: string) {
    if (expandedProductId === productId) {
      setExpandedProductId(null)
      return
    }
    setExpandedProductId(productId)
    setLoadingSkus(true)
    try {
      const detail = await pmsApi.getProduct(token ?? '', productId)
      setSkus(detail.variants.flatMap(v => v.skus ?? []))
    } finally {
      setLoadingSkus(false)
    }
  }

  return (
    <div className="rounded-lg border border-line bg-surface p-3 space-y-2">
      <div className="flex items-center gap-2">
        <div className="relative flex-1">
          <Search size={12} className="absolute left-2.5 top-1/2 -translate-y-1/2 text-ink-4" />
          <input
            autoFocus value={query} onChange={e => setQuery(e.target.value)}
            placeholder={t('ui:skuPicker.searchPlaceholder')}
            className="w-full pl-7 pr-2 py-1.5 text-xs rounded-lg bg-surface border border-line text-ink outline-none focus:ring-1 focus:ring-focus"
          />
        </div>
        <button aria-label={t('common:actions.close')} type="button" onClick={onClose} title={t('common:actions.close')}>
          <X size={14} className="text-ink-4" />
        </button>
      </div>
      <div className="max-h-48 overflow-auto space-y-1">
        {products.length === 0 && <p className="text-2xs text-ink-4 px-1">{t('ui:skuPicker.noProducts')}</p>}
        {products.map(p => (
          <div key={p.id}>
            <button
              type="button"
              onClick={() => toggleProduct(p.id)}
              className="w-full text-left px-2 py-1.5 text-xs rounded hover:bg-hover text-ink"
            >
              {p.name}
            </button>
            {expandedProductId === p.id && (
              <div className="pl-4 space-y-0.5">
                {loadingSkus && <p className="text-2xs text-ink-4">{t('common:status.loading')}</p>}
                {!loadingSkus && skus.length === 0 && <p className="text-2xs text-ink-4">{t('ui:skuPicker.noSkus')}</p>}
                {!loadingSkus && skus.map(s => (
                  <button
                    key={s.id}
                    type="button"
                    onClick={() => onSelect(s, `${p.name} — ${s.sku_code}`, p.name)}
                    className="w-full flex items-center justify-between text-left px-2 py-1 text-2xs rounded hover:bg-hover"
                  >
                    <span className="font-mono text-ink-2">{s.sku_code}</span>
                    <span className="text-ink-4">{(s.price_minor_units / 100).toFixed(2)} {s.currency_code}</span>
                  </button>
                ))}
              </div>
            )}
          </div>
        ))}
      </div>
    </div>
  )
}
