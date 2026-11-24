import { test, expect, api } from './fixtures'

// The SKU stock page: pick a place, add one, take it back off. Net zero on
// the shared test data. Screenshots of the page per look and appearance.
interface ProductRow { id: string; name: string }
interface ProductDetail { variants: { skus?: { id: string }[] }[] }

async function carneSecaSku(request: Parameters<typeof api>[0], token: string) {
  const products = await api<ProductRow[]>(request, token, '/pms/products?search=Carne%20Seca&per_page=5', { method: 'GET' })
  const product = products[0]
  if (!product) throw new Error('fixture product "Carne Seca" missing')
  const detail = await api<ProductDetail>(request, token, `/pms/products/${product.id}`, { method: 'GET' })
  const sku = detail.variants.flatMap(v => v.skus ?? [])[0]
  return { productId: product.id, skuId: sku.id }
}

test('adds one unit and removes it again from the stock page', async ({ page, request, token }) => {
  const { productId, skuId } = await carneSecaSku(request, token)
  await page.goto(`/inventory/products/${productId}/skus/${skuId}/stock`)
  await expect(page.getByRole('heading', { name: 'Adjust stock' })).toBeVisible()

  const total = page.getByTestId('stock-total')
  await expect(total).toBeVisible()
  const before = await total.textContent()

  await page.getByRole('button', { name: 'Correction' }).click()
  await page.getByLabel('Quantity').fill('1')
  await page.getByLabel('Note (optional)').fill('e2e adjust')
  await page.getByRole('button', { name: 'Record change' }).click()
  await expect(page.getByRole('status')).toContainText('Recorded')
  await expect(page.getByText(/Correction — e2e adjust/).first()).toBeVisible()

  await page.getByRole('button', { name: 'Remove' }).click()
  await page.getByRole('button', { name: 'Correction' }).click()
  await page.getByLabel('Quantity').fill('1')
  await page.getByRole('button', { name: 'Record change' }).click()
  await expect(page.getByRole('status')).toContainText('Recorded')
  await expect(total).toHaveText(before ?? '')
})

for (const [look, appearance] of [['wallet', 'light'], ['wallet', 'dark'], ['graphite', 'light'], ['graphite', 'dark']] as const) {
  test.describe(`stock page, ${look} ${appearance}`, () => {
    test.use({ look, appearance })
    test('screenshot', async ({ page, request, token }) => {
      const { productId, skuId } = await carneSecaSku(request, token)
      await page.goto(`/inventory/products/${productId}/skus/${skuId}/stock`)
      await expect(page.getByRole('heading', { name: 'Adjust stock' })).toBeVisible()
      await page.getByLabel('Quantity').fill('2')
      await page.waitForTimeout(500)
      await page.screenshot({ path: `e2e-screenshots/${look}-${appearance}-adjust-stock.png`, fullPage: true })
    })
  })
}

test('the modal opens the full page', async ({ page, request, token }) => {
  const { productId, skuId } = await carneSecaSku(request, token)
  await page.goto(`/inventory/products/${productId}/skus/${skuId}/adjust?open=${productId}`)
  const dialog = page.getByRole('dialog')
  await expect(dialog).toBeVisible()
  await page.screenshot({ path: 'e2e-screenshots/wallet-light-adjust-modal.png' })
  await dialog.getByRole('button', { name: 'Open full page' }).click()
  await expect(page).toHaveURL(/\/stock/)
  await expect(page.getByRole('heading', { name: 'Adjust stock' })).toBeVisible()
})
