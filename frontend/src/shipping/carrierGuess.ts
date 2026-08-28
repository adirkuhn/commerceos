// shipping_method_title/shipping_method are merchant-typed WooCommerce
// fields (see backend domain/oms/sales_order.go's doc comment) -- never a
// controlled vocabulary, so a match here is only a best-effort guess. It's
// used solely to pre-select AddressConfirmModal's carrier dropdown; staff
// can always override it before creating the label.
const KEYWORD_CARRIER: { keyword: string; carrierId: number }[] = [
  { keyword: 'postnl', carrierId: 1 },
  { keyword: 'bpost', carrierId: 2 },
  { keyword: 'dpd', carrierId: 4 },
  { keyword: 'dhl', carrierId: 9 },
  { keyword: 'ups', carrierId: 12 },
]

export function guessCarrierFromShippingMethod(
  shippingMethodTitle: string | undefined,
  shippingMethod: string | undefined,
  fallback: number,
): number {
  const text = `${shippingMethodTitle ?? ''} ${shippingMethod ?? ''}`.toLowerCase()
  const match = KEYWORD_CARRIER.find(({ keyword }) => text.includes(keyword))
  return match?.carrierId ?? fallback
}
