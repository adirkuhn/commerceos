import { useEffect, useState } from 'react'
import { getCustomer } from '../../customers/api'
import type { Customer } from '../../customers/types'
import { listPriceList } from '../../pricelist/api'
import type { PriceListRow } from '../../pricelist/types'
import { draftFromOrder } from '../create/editDraft'
import type { OrderDraft } from '../create/orderDraft'
import type { SalesOrderDetail } from '../types'

// The order form's starting point for editing an order: its lines at
// their products' current VAT rate and stock (looked up in the price list
// by SKU code), and its customer. Null draft until loaded; error is the
// failed request, for the page to word.
export function useEditDraft(token: string, order: SalesOrderDetail | undefined) {
  const [draft, setDraft] = useState<OrderDraft | null>(null)
  const [customer, setCustomer] = useState<Customer | null>(null)
  const [error, setError] = useState<unknown>(null)

  useEffect(() => {
    if (!order) return
    let cancelled = false
    const codes = [...new Set(order.line_items.filter(li => li.sku_id).map(li => li.sku_code))]
    Promise.all([
      Promise.all(codes.map(code => listPriceList(token, { search: code, perPage: 20 }).then(p => p.rows))),
      order.sales_order.customer_id ? getCustomer(token, order.sales_order.customer_id) : Promise.resolve(null),
    ]).then(([pages, c]) => {
      if (cancelled) return
      const rows = new Map<string, PriceListRow>(pages.flat().map(r => [r.sku_id, r]))
      setDraft(draftFromOrder(order, rows))
      setCustomer(c)
    }).catch(err => {
      if (!cancelled) setError(err)
    })
    return () => { cancelled = true }
  }, [token, order])

  return { draft, customer, error }
}
