import { useMemo, useState } from 'react'
import { Plus, Trash2, ShoppingCart, AlertTriangle } from 'lucide-react'
import { Seo } from '@/components/seo/Seo'
import { PageHeader } from '@/components/layout/PageHeader'
import { Select } from '@/components/ui/Select'
import { Input } from '@/components/ui/Input'
import { Button } from '@/components/ui/Button'
import { Separator } from '@/components/ui/Separator'
import { LoadingState } from '@/components/ui/States'
import { formatMoney } from '@/lib/utils'
import { applyPriceList, priceForQuantity } from '@/lib/b2bPricing'
import { useProducts } from '@/features/products/hooks'
import { useAddCartItem } from '@/features/cart/hooks'
import { useAuthStore } from '@/stores/authStore'
import { PRICE_LISTS } from '@/config/demoData'
import { toast } from '@/components/ui/toast'
import type { Product } from '@/types'

interface OrderRow {
  rowId: string
  productId: string
  quantity: number
}

let rowSeq = 0
function newRow(): OrderRow {
  rowSeq += 1
  return { rowId: `row-${rowSeq}`, productId: '', quantity: 1 }
}

// Quick order / bulk order -- its own route, deliberately not bolted onto
// the regular product grid or cart page: a B2B buyer knows the SKUs they
// want and just needs a fast grid to enter quantities and add everything
// to cart in one action, not to browse.
export default function AccountQuickOrderPage() {
  const { data, isLoading } = useProducts({})
  const products = useMemo(() => data?.items ?? [], [data])
  const addItem = useAddCartItem()
  const company = useAuthStore(s => s.session?.customer.company)
  const priceList = company?.priceListId ? PRICE_LISTS.find(pl => pl.id === company.priceListId) : undefined

  const [rows, setRows] = useState<OrderRow[]>([newRow(), newRow(), newRow()])

  const productById = useMemo(() => new Map(products.map(p => [p.id, p])), [products])

  function updateRow(rowId: string, patch: Partial<OrderRow>) {
    setRows(prev => prev.map(r => (r.rowId === rowId ? { ...r, ...patch } : r)))
  }

  function removeRow(rowId: string) {
    setRows(prev => prev.filter(r => r.rowId !== rowId))
  }

  function unitPriceFor(product: Product, quantity: number): number {
    const withPriceList = applyPriceList(product.price, priceList)
    return priceForQuantity(withPriceList, quantity, product.quantityBreaks)
  }

  const lineItems = rows
    .map(row => ({ row, product: productById.get(row.productId) }))
    .filter((li): li is { row: OrderRow; product: Product } => !!li.product && li.row.quantity > 0)

  const belowMoqRows = lineItems.filter(li => li.product.minOrderQuantity && li.row.quantity < li.product.minOrderQuantity)
  const grandTotal = lineItems.reduce((sum, li) => sum + unitPriceFor(li.product, li.row.quantity) * li.row.quantity, 0)
  const canSubmit = lineItems.length > 0 && belowMoqRows.length === 0

  async function addAllToCart() {
    for (const li of lineItems) {
      await addItem.mutateAsync({
        productId: li.product.id, name: li.product.name, sku: li.product.sku,
        imageUrl: li.product.images[0]?.url, price: unitPriceFor(li.product, li.row.quantity), quantity: li.row.quantity,
      })
    }
    toast({ title: `Added ${lineItems.length} item${lineItems.length === 1 ? '' : 's'} to cart`, variant: 'success' })
    setRows([newRow(), newRow(), newRow()])
  }

  return (
    <>
      <Seo title="Quick order" />
      <PageHeader title="Quick order" description="Enter SKUs and quantities to add multiple products to your cart at once." />

      {isLoading ? <LoadingState /> : (
        <div className="space-y-6">
          <div className="overflow-x-auto rounded-[var(--radius)] border border-border">
            <table className="w-full text-left text-sm">
              <thead className="border-b border-border bg-secondary/50 text-xs uppercase tracking-wide text-muted-foreground">
                <tr>
                  <th className="px-4 py-3 font-medium">Product</th>
                  <th className="px-4 py-3 font-medium">Quantity</th>
                  <th className="px-4 py-3 text-right font-medium">Unit price</th>
                  <th className="px-4 py-3 text-right font-medium">Line total</th>
                  <th className="px-4 py-3" />
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {rows.map(row => {
                  const product = productById.get(row.productId)
                  const belowMoq = !!product?.minOrderQuantity && row.quantity < product.minOrderQuantity
                  return (
                    <tr key={row.rowId}>
                      <td className="px-4 py-2">
                        <Select
                          aria-label="Product"
                          value={row.productId}
                          onChange={(e) => updateRow(row.rowId, { productId: e.target.value })}
                        >
                          <option value="">Select a product…</option>
                          {products.map(p => <option key={p.id} value={p.id}>{p.name} ({p.sku})</option>)}
                        </Select>
                      </td>
                      <td className="px-4 py-2">
                        <Input
                          type="number" min={1} aria-label="Quantity" value={row.quantity} className="w-24"
                          onChange={(e) => updateRow(row.rowId, { quantity: Math.max(0, Number(e.target.value) || 0) })}
                        />
                        {belowMoq && (
                          <p className="mt-1 flex items-center gap-1 text-xs text-destructive">
                            <AlertTriangle className="h-3 w-3" /> MOQ is {product!.minOrderQuantity}
                          </p>
                        )}
                      </td>
                      <td className="px-4 py-2 text-right">{product ? formatMoney(unitPriceFor(product, row.quantity), product.currency) : '—'}</td>
                      <td className="px-4 py-2 text-right font-medium">{product ? formatMoney(unitPriceFor(product, row.quantity) * row.quantity, product.currency) : '—'}</td>
                      <td className="px-4 py-2 text-right">
                        <button type="button" aria-label="Remove row" onClick={() => removeRow(row.rowId)} className="text-muted-foreground hover:text-destructive">
                          <Trash2 className="h-4 w-4" />
                        </button>
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>

          <Button type="button" variant="outline" size="sm" onClick={() => setRows(prev => [...prev, newRow()])}>
            <Plus className="mr-2 h-4 w-4" /> Add row
          </Button>

          <Separator />

          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-muted-foreground">{lineItems.length} line item{lineItems.length === 1 ? '' : 's'}{priceList ? ` · ${priceList.name} pricing applied` : ''}</p>
              <p className="text-lg font-semibold">{formatMoney(grandTotal, 'USD')}</p>
            </div>
            <Button type="button" disabled={!canSubmit || addItem.isPending} onClick={addAllToCart}>
              <ShoppingCart className="mr-2 h-4 w-4" /> {addItem.isPending ? 'Adding…' : 'Add all to cart'}
            </Button>
          </div>
        </div>
      )}
    </>
  )
}
