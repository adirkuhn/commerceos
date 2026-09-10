// @vitest-environment jsdom
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'
import { loyaltyPointsPlugin } from '../loyaltyPointsPlugin'
import { giftWrapPlugin } from '../giftWrapPlugin'
import { PRODUCTS } from '@/config/demoData'

describe('loyaltyPointsPlugin', () => {
  it('renders a product tab computing 1 point per dollar', () => {
    const product = PRODUCTS.find(p => p.price === 4900) ?? PRODUCTS[0]
    const tab = loyaltyPointsPlugin.productTabs![0]
    const Component = tab.render(product)!
    render(<Component />)

    const points = Math.floor(product.price / 100)
    expect(screen.getByText(new RegExp(`${points} points`))).toBeInTheDocument()
  })
})

describe('giftWrapPlugin', () => {
  it('renders an interactive checkout-step checkbox registered on the shipping step', async () => {
    const user = userEvent.setup()
    expect(giftWrapPlugin.checkoutExtensions![0].step).toBe('shipping')
    const Extension = giftWrapPlugin.checkoutExtensions![0].render
    render(<Extension />)

    const checkbox = screen.getByRole('checkbox', { name: /Add gift wrap/ })
    expect(checkbox).not.toBeChecked()
    await user.click(checkbox)
    expect(checkbox).toBeChecked()
  })
})
