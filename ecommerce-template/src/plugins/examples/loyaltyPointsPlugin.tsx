import type { EcommercePlugin, Product } from '@/types'
import { formatMoney } from '@/lib/utils'

// A real plugin worth demoing: a product-page tab computed from the
// product itself (1 point per dollar), and a hook that would let a
// points ledger react to real purchases. Proves EcommercePlugin's
// productTabs + hooks mechanisms end-to-end, not just the type shape.
function LoyaltyPointsTab({ product }: { product: Product }) {
  const points = Math.floor(product.price / 100)
  return (
    <div className="space-y-2 text-sm text-muted-foreground">
      <p>
        Earn <strong className="text-foreground">{points} points</strong> ({formatMoney(product.price, product.currency)}) when you buy this item.
      </p>
      <p>Redeem 500 points for a $5 reward at checkout.</p>
    </div>
  )
}

export const loyaltyPointsPlugin: EcommercePlugin = {
  name: 'loyalty-points',
  productTabs: [
    {
      id: 'loyalty-points',
      label: 'Rewards',
      // render closes over the given product and returns a zero-props
      // component, matching ProductTab's `(product) => ComponentType`
      // contract (see types/plugin.ts).
      render: (product) => () => <LoyaltyPointsTab product={product} />,
    },
  ],
  hooks: {
    order_placed: (payload) => {
      // eslint-disable-next-line no-console
      console.info('[loyalty-points] order placed, points would be credited here', payload)
    },
  },
}
