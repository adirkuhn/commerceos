// @vitest-environment jsdom
import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, beforeEach } from 'vitest'
import AccountQuickOrderPage from '../AccountQuickOrderPage'
import { renderWithProviders } from '@/test/testUtils'
import { useAuthStore } from '@/stores/authStore'
import { useCartStore } from '@/stores/cartStore'
import { DEMO_CUSTOMER, PRODUCTS } from '@/config/demoData'
import { cartRepository } from '@/repositories'
import { applyPriceList, priceForQuantity } from '@/lib/b2bPricing'
import { PRICE_LISTS } from '@/config/demoData'
import { formatMoney } from '@/lib/utils'

const wholesaleTee = PRODUCTS.find(p => p.name === 'Meridian Everyday Tee')!
const plainProduct = PRODUCTS.find(p => !p.minOrderQuantity && !p.quantityBreaks)!

function loginAsCompanyCustomer() {
  useAuthStore.setState({
    isAuthenticated: true,
    session: { customer: DEMO_CUSTOMER, token: 'test-token', expiresAt: '2099-01-01T00:00:00Z', twoFactorVerified: true },
  })
}

beforeEach(() => {
  useAuthStore.setState({ session: null, isAuthenticated: false })
  useCartStore.setState({ cartId: `quick-order-test-${Math.random()}`, isDrawerOpen: false })
})

describe('AccountQuickOrderPage', () => {
  it('computes price-list + quantity-break pricing for a selected row', async () => {
    loginAsCompanyCustomer()
    const user = userEvent.setup()
    renderWithProviders(<AccountQuickOrderPage />)

    await screen.findByText(/0 line items/)

    await user.selectOptions(screen.getAllByLabelText('Product')[0], wholesaleTee.id)
    const quantityInput = screen.getAllByLabelText('Quantity')[0]
    await user.clear(quantityInput)
    await user.type(quantityInput, '50')

    const priceList = PRICE_LISTS.find(pl => pl.id === DEMO_CUSTOMER.company!.priceListId)
    const expectedUnit = priceForQuantity(applyPriceList(wholesaleTee.price, priceList), 50, wholesaleTee.quantityBreaks)

    await waitFor(() => {
      expect(screen.getByText(formatMoney(expectedUnit, wholesaleTee.currency))).toBeInTheDocument()
    })
    expect(screen.getByText(/1 line item/)).toBeInTheDocument()
    expect(screen.getByText(/Wholesale pricing applied/)).toBeInTheDocument()
  })

  it('blocks submission and shows a warning when quantity is below MOQ', async () => {
    loginAsCompanyCustomer()
    const user = userEvent.setup()
    renderWithProviders(<AccountQuickOrderPage />)
    await screen.findByText(/0 line items/)

    await user.selectOptions(screen.getAllByLabelText('Product')[0], wholesaleTee.id)
    const quantityInput = screen.getAllByLabelText('Quantity')[0]
    await user.clear(quantityInput)
    await user.type(quantityInput, '3')

    expect(await screen.findByText(`MOQ is ${wholesaleTee.minOrderQuantity}`)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Add all to cart/ })).toBeDisabled()
  })

  it('adds every valid row to the cart at the computed unit price', async () => {
    loginAsCompanyCustomer()
    const user = userEvent.setup()
    renderWithProviders(<AccountQuickOrderPage />)
    await screen.findByText(/0 line items/)

    await user.selectOptions(screen.getAllByLabelText('Product')[0], plainProduct.id)
    const qty1 = screen.getAllByLabelText('Quantity')[0]
    await user.clear(qty1)
    await user.type(qty1, '3')

    await user.selectOptions(screen.getAllByLabelText('Product')[1], wholesaleTee.id)
    const qty2 = screen.getAllByLabelText('Quantity')[1]
    await user.clear(qty2)
    await user.type(qty2, String(wholesaleTee.minOrderQuantity))

    await user.click(await screen.findByRole('button', { name: /Add all to cart/ }))

    await waitFor(async () => {
      const cart = await cartRepository.get(useCartStore.getState().cartId)
      expect(cart.items).toHaveLength(2)
    })
    const cart = await cartRepository.get(useCartStore.getState().cartId)
    const teeLine = cart.items.find(i => i.productId === wholesaleTee.id)!
    const priceList = PRICE_LISTS.find(pl => pl.id === DEMO_CUSTOMER.company!.priceListId)
    expect(teeLine.price).toBe(priceForQuantity(applyPriceList(wholesaleTee.price, priceList), wholesaleTee.minOrderQuantity!, wholesaleTee.quantityBreaks))
  })
})
