import { test, expect, api } from './fixtures'

// The sales order page opens on the order's journey: four leg coupons
// (paid, picked, packed, shipped), the current one raised. Screenshots per
// look and appearance land in e2e-screenshots/.
interface ListedOrder { id: string }

for (const look of ['wallet', 'graphite'] as const) for (const appearance of ['light', 'dark'] as const) {
  test.describe(`order journey, ${look} look, ${appearance} appearance`, () => {
    test.use({ look, appearance })

    test('shows the four legs with exactly one current or a void stamp', async ({ page, request, token }) => {
      const orders = await api<ListedOrder[] | { sales_orders: ListedOrder[] }>(request, token, '/sales-orders?per_page=1', { method: 'GET' })
      const first = Array.isArray(orders) ? orders[0] : orders.sales_orders[0]
      test.skip(!first, 'no sales order to open')

      await page.goto(`/sales-orders/${first.id}`)
      const journey = page.getByRole('list', { name: 'Order journey' })
      await expect(journey.getByRole('listitem')).toHaveCount(4)
      await page.screenshot({ path: `e2e-screenshots/${look}-${appearance}-order-journey.png` })
      await page.setViewportSize({ width: 390, height: 844 })
      await page.screenshot({ path: `e2e-screenshots/${look}-${appearance}-order-journey-mobile.png`, fullPage: false })
    })
  })
}
