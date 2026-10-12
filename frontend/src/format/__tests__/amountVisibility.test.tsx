// @vitest-environment jsdom
import { act, fireEvent, render, renderHook, screen } from '@testing-library/react'
import type { ReactNode } from 'react'
import { describe, expect, it } from 'vitest'
import { AmountVisibilityProvider, Money, maskMoney, useAmountVisibility, useMoneyFormat } from '../amountVisibility'
import { formatMoney } from '../money'

const hidden = ({ children }: { children: ReactNode }) => <AmountVisibilityProvider>{children}</AmountVisibilityProvider>

describe('maskMoney', () => {
  it('keeps the currency and hides every digit', () => {
    const masked = maskMoney('EUR')
    expect(masked).toContain('€')
    expect(masked).toContain('••••')
    expect(masked).not.toMatch(/\d/)
  })

  it('looks the same whatever the amount, so the size does not leak', () => {
    expect(maskMoney('USD')).toBe(maskMoney('USD'))
  })

  it('falls back to the code for a currency Intl does not know', () => {
    expect(maskMoney('XYZ1')).toBe('•••• XYZ1')
  })
})

describe('Money', () => {
  it('shows the amount when no privacy provider is mounted (tests, standalone screens)', () => {
    render(<Money minor={1250} currency="EUR" />)
    expect(screen.getByText(formatMoney(1250, 'EUR'))).toBeTruthy()
  })

  it('hides the amount by default inside the app', () => {
    render(<Money minor={1250} currency="EUR" />, { wrapper: hidden })
    expect(screen.queryByText(formatMoney(1250, 'EUR'))).toBeNull()
    expect(screen.getByRole('button', { name: 'Show amount' }).textContent).toBe(maskMoney('EUR'))
  })

  it('reveals one amount on click and hides it again on a second click', () => {
    render(<><Money minor={1250} currency="EUR" /><Money minor={990} currency="EUR" /></>, { wrapper: hidden })
    const [first] = screen.getAllByRole('button', { name: 'Show amount' })

    fireEvent.click(first)
    expect(screen.getByText(formatMoney(1250, 'EUR'))).toBeTruthy()
    expect(screen.queryByText(formatMoney(990, 'EUR'))).toBeNull()

    fireEvent.click(screen.getByRole('button', { pressed: true }))
    expect(screen.queryByText(formatMoney(1250, 'EUR'))).toBeNull()
  })

  it('does not trigger a clickable row around it', () => {
    let rowClicks = 0
    render(<div onClick={() => { rowClicks++ }}><Money minor={1250} currency="EUR" /></div>, { wrapper: hidden })
    fireEvent.click(screen.getByRole('button', { name: 'Show amount' }))
    expect(rowClicks).toBe(0)
  })

  it('shows every amount once all are revealed', () => {
    function Harness() {
      const { setVisible } = useAmountVisibility()
      return <><button onClick={() => setVisible(true)}>all</button><Money minor={1250} currency="EUR" /></>
    }
    render(<Harness />, { wrapper: hidden })
    fireEvent.click(screen.getByText('all'))
    expect(screen.getByText(formatMoney(1250, 'EUR'))).toBeTruthy()
    expect(screen.queryByRole('button', { name: 'Show amount' })).toBeNull()
  })
})

describe('useMoneyFormat', () => {
  it('masks text amounts (chart tooltips, labels) while hidden', async () => {
    const { result } = renderHook(() => ({ fmt: useMoneyFormat(), vis: useAmountVisibility() }), { wrapper: hidden })
    expect(result.current.fmt(1250, 'EUR')).toBe(maskMoney('EUR'))

    await act(async () => { result.current.vis.setVisible(true) })

    expect(result.current.fmt(1250, 'EUR')).toBe(formatMoney(1250, 'EUR'))
  })

  it('formats normally without a provider', () => {
    const { result } = renderHook(() => useMoneyFormat())
    expect(result.current(1250, 'EUR')).toBe(formatMoney(1250, 'EUR'))
  })
})
