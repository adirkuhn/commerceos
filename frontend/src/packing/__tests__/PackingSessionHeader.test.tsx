// @vitest-environment jsdom
import { render, screen } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { PackingSessionHeader } from '../PackingSessionHeader'
import type { SalesOrder } from '../../oms/types'

function order(overrides: Partial<SalesOrder> = {}): SalesOrder {
  return {
    id: 'so-1', tenant_id: 't', source: 'woocommerce', external_order_id: 'ext-1', number: '25990', order_key: '',
    status: 'processing', currency_code: 'EUR', billing: { first_name: 'Ann', last_name: 'Lee' },
    shipping: { address_1: 'Sloep 15', postcode: '1276 CT', city: 'Huizen', country: 'NL' },
    total_minor_units: 1250,
    ...overrides,
  } as SalesOrder
}

function renderHeader(o: SalesOrder) {
  render(
    <PackingSessionHeader
      order={o} showCreateLabel={false} readOnly={false}
      onCreateLabel={vi.fn()} onAddItem={vi.fn()} onStartOver={vi.fn()} onComplete={vi.fn()}
    />,
  )
}

describe('PackingSessionHeader', () => {
  it('shows the order, customer and shipping address', () => {
    renderHeader(order())

    expect(screen.getByText('#25990')).toBeInTheDocument()
    expect(screen.getByText('Ann Lee')).toBeInTheDocument()
    expect(screen.getByText('Sloep 15, 1276 CT Huizen, NL')).toBeInTheDocument()
  })

  it('shows the shipping method from the sales order', () => {
    renderHeader(order({ shipping_method_title: 'PostNL pakket' }))

    expect(screen.getByTitle('Shipping method')).toHaveTextContent('PostNL pakket')
  })

  it('shows no shipping method line when the order has none', () => {
    renderHeader(order())

    expect(screen.queryByTitle('Shipping method')).not.toBeInTheDocument()
  })
})
