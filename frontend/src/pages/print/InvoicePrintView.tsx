import { useEffect } from 'react'
import { X } from 'lucide-react'
import type { SalesOrder, SalesOrderLineItem, Address } from '../../oms/types'
import type { CompanyProfile } from '../../settings/company/types'
import { formatMoney } from '../../format/money'

function addressBlock(address?: Address): string[] {
  if (!address) return []
  const name = [address.first_name, address.last_name].filter(Boolean).join(' ')
  return [
    name,
    address.company,
    address.address_1,
    address.address_2,
    [address.city, address.state, address.postcode].filter(Boolean).join(', '),
    address.country,
  ].filter((l): l is string => Boolean(l))
}

// The seller's lines as printed on the invoice; empty values are left out.
function sellerBlock(seller: CompanyProfile): string[] {
  const a = seller.address
  return [
    seller.legal_name,
    [a.street, a.house_number].filter(Boolean).join(' '),
    [a.postal_code, a.city].filter(Boolean).join(' '),
    seller.vat_number && `VAT: ${seller.vat_number}`,
    seller.chamber_of_commerce && `Chamber of Commerce: ${seller.chamber_of_commerce}`,
    seller.iban && `IBAN: ${seller.iban}`,
    seller.email,
    seller.phone,
  ].filter((l): l is string => Boolean(l))
}

interface InvoicePrintViewProps {
  order: SalesOrder
  lineItems: SalesOrderLineItem[]
  // The tenant's company profile; null when it hasn't been filled in.
  seller: CompanyProfile | null
  onClose: () => void
}

// Plain-HTML printable view, same zero-dependency approach as
// PickingListPrintView -- no PDF library, just Tailwind print: variants
// and the browser's own print dialog.
export function InvoicePrintView({ order, lineItems, seller, onClose }: InvoicePrintViewProps) {
  useEffect(() => { window.print() }, [])

  const billing = addressBlock(order.billing)
  const shipping = addressBlock(order.shipping)

  return (
    <div className="fixed inset-0 z-50 bg-white overflow-y-auto">
      <button aria-label="Close"
        onClick={onClose}
        className="print:hidden fixed top-4 right-4 w-9 h-9 flex items-center justify-center rounded-lg bg-gray-100 hover:bg-gray-200 text-gray-600"
        title="Close"
      >
        <X size={16} />
      </button>
      <div className="max-w-2xl mx-auto p-10 text-gray-900">
        <div className="flex items-start justify-between mb-8">
          <div>
            <h1 className="text-2xl font-bold">Invoice</h1>
            {seller && seller.legal_name ? (
              <section aria-label="Seller" className="mt-3 text-sm">
                {sellerBlock(seller).map((l, i) => <p key={i} className={i === 0 ? 'font-medium' : 'text-gray-700'}>{l}</p>)}
              </section>
            ) : (
              <p className="print:hidden mt-3 text-sm text-amber-800 bg-amber-50 rounded-lg px-3 py-2">
                This invoice has no seller details. Add your company details in Settings → Company.
              </p>
            )}
          </div>
          <div className="text-right text-sm">
            <p className="font-medium">#{order.number || order.external_order_id}</p>
            <p className="text-gray-600">{new Date(order.occurred_at).toLocaleDateString()}</p>
          </div>
        </div>

        <div className="grid grid-cols-2 gap-6 mb-8 text-sm">
          <div>
            <p className="font-medium mb-1">Billing</p>
            {billing.length === 0 ? <p className="text-gray-500">—</p> : billing.map((l, i) => <p key={i}>{l}</p>)}
          </div>
          <div>
            <p className="font-medium mb-1">Shipping</p>
            {shipping.length === 0 ? <p className="text-gray-500">—</p> : shipping.map((l, i) => <p key={i}>{l}</p>)}
          </div>
        </div>

        <table className="w-full text-sm border-collapse">
          <thead>
            <tr className="border-b-2 border-gray-900">
              <th className="text-left py-2">Product</th>
              <th className="text-right py-2">Qty</th>
              <th className="text-right py-2">Price</th>
              <th className="text-right py-2">Total</th>
            </tr>
          </thead>
          <tbody>
            {lineItems.map(li => (
              <tr key={li.id} className="border-b border-gray-300">
                <td className="py-2.5">{li.name}</td>
                <td className="py-2.5 text-right">{li.quantity}</td>
                <td className="py-2.5 text-right">{formatMoney(li.price_minor_units, order.currency_code)}</td>
                <td className="py-2.5 text-right font-medium">{formatMoney(li.total_minor_units, order.currency_code)}</td>
              </tr>
            ))}
          </tbody>
        </table>

        <div className="flex justify-end mt-4">
          <div className="w-56 space-y-1 text-sm">
            <div className="flex justify-between text-gray-600">
              <span>Discount</span>
              <span>{formatMoney(order.discount_total_minor_units, order.currency_code)}</span>
            </div>
            <div className="flex justify-between text-gray-600">
              <span>Shipping</span>
              <span>{formatMoney(order.shipping_total_minor_units, order.currency_code)}</span>
            </div>
            <div className="flex justify-between text-gray-600">
              <span>Tax</span>
              <span>{formatMoney(order.tax_total_minor_units, order.currency_code)}</span>
            </div>
            <div className="flex justify-between font-bold pt-1 border-t border-gray-900">
              <span>Total</span>
              <span>{formatMoney(order.total_minor_units, order.currency_code)}</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
