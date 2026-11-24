// @vitest-environment jsdom
import { render, screen } from '@testing-library/react'
import { describe, it, expect } from 'vitest'
import { ExpiryStamp } from '../ExpiryStamp'

const inDays = (n: number) => new Date(Date.now() + n * 86_400_000).toISOString()

describe('ExpiryStamp', () => {
  it('stamps an expired lot in the danger colour and says so', () => {
    render(<ExpiryStamp date={inDays(-2)} />)
    const stamp = screen.getByText(/Expired/)
    expect(stamp.closest('[data-expiry]')).toHaveAttribute('data-expiry', 'expired')
  })

  it('stamps a lot expiring within 30 days as soon', () => {
    render(<ExpiryStamp date={inDays(10)} />)
    expect(screen.getByText(/Expires soon/).closest('[data-expiry]')).toHaveAttribute('data-expiry', 'soon')
  })

  it('shows a later date quietly, without a state word', () => {
    render(<ExpiryStamp date={inDays(90)} />)
    const stamp = document.querySelector('[data-expiry]')
    expect(stamp).toHaveAttribute('data-expiry', 'ok')
    expect(screen.queryByText(/Expired|Expires soon/)).toBeNull()
  })

  it('renders nothing without a date', () => {
    const { container } = render(<ExpiryStamp date={null} />)
    expect(container).toBeEmptyDOMElement()
  })
})
