import type { Customer } from '../customers/types'
import type { Address, UpdateSalesOrderInput } from './types'

const hasAny = (a?: Address) => !!a && Object.values(a).some(v => typeof v === 'string' && v.trim() !== '')

/**
 * The order update for a new customer: the link, plus the customer's billing
 * and shipping address when it has one (an empty one keeps the order's).
 * null removes the customer ("" clears customer_id on the API).
 */
export function orderCustomerChange(customer: Customer | null): UpdateSalesOrderInput {
  if (!customer) return { customer_id: '' }
  const change: UpdateSalesOrderInput = { customer_id: customer.id }
  if (hasAny(customer.billing)) change.billing = customer.billing
  if (hasAny(customer.shipping)) change.shipping = customer.shipping
  return change
}
