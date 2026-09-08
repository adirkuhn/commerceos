import { useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { getProduct } from '../../api/products'
import { useCart } from '../../cart/CartContext'
import { formatMoney } from '../../format'
import type { StorefrontProduct } from '../../types'

export function ProductDetailsBlock() {
  const { id } = useParams<{ id: string }>()
  const { addItem } = useCart()
  const [product, setProduct] = useState<StorefrontProduct | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [addedSku, setAddedSku] = useState<string | null>(null)

  useEffect(() => {
    if (!id) {
      setLoading(false)
      return
    }
    let cancelled = false
    setLoading(true)
    getProduct(id)
      .then(p => { if (!cancelled) setProduct(p) })
      .catch(e => { if (!cancelled) setError(e instanceof Error ? e.message : 'Unknown error') })
      .finally(() => { if (!cancelled) setLoading(false) })
    return () => { cancelled = true }
  }, [id])

  if (!id) return <p className="p-4 text-sm text-red-600">No product selected.</p>
  if (loading) return <p className="p-4 text-sm text-gray-500">Loading product…</p>
  if (error) return <p className="p-4 text-sm text-red-600">Could not load this product: {error}</p>
  if (!product) return null

  const handleAdd = async (skuCode: string) => {
    setAddedSku(null)
    await addItem(skuCode, 1)
    setAddedSku(skuCode)
  }

  return (
    <div className="h-full overflow-auto p-6">
      <h1 className="text-2xl font-semibold text-brand-700">{product.name}</h1>
      {product.description && <p className="mt-2 text-brand-500">{product.description}</p>}
      <div className="mt-6 space-y-4">
        {product.variants.map(variant => (
          <div key={variant.id}>
            <p className="text-sm font-medium text-brand-700">{variant.name}</p>
            <div className="mt-2 space-y-2">
              {variant.skus.map(sku => (
                <div key={sku.id} className="flex items-center justify-between rounded-lg border border-gray-200 p-3">
                  <div>
                    <p className="text-sm text-brand-700">{sku.sku_code}</p>
                    <p className="text-sm text-brand-500">
                      {sku.sale_price_minor_units != null ? (
                        <>
                          <span className="mr-1 text-gray-400 line-through">{formatMoney(sku.price_minor_units, sku.currency_code)}</span>
                          {formatMoney(sku.sale_price_minor_units, sku.currency_code)}
                        </>
                      ) : formatMoney(sku.price_minor_units, sku.currency_code)}
                    </p>
                  </div>
                  <button
                    onClick={() => handleAdd(sku.sku_code)}
                    disabled={sku.stock_quantity <= 0}
                    className="rounded-lg bg-brand-600 px-3 py-1.5 text-xs font-medium text-white transition-colors hover:bg-brand-700 disabled:opacity-50"
                  >
                    {sku.stock_quantity <= 0 ? 'Out of stock' : addedSku === sku.sku_code ? 'Added' : 'Add to cart'}
                  </button>
                </div>
              ))}
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
