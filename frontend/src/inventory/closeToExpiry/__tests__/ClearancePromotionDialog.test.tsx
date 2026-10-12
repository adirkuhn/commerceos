// @vitest-environment jsdom
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { ClearancePromotionDialog } from '../ClearancePromotionDialog'
import * as promotionsApi from '../../../promotions/api'
import { CodedApiError } from '../../../api/codedApiError'
import type { CloseToExpiryBatch } from '../types'
import { zonedMidnight } from '../../../format/calendarDay'

vi.mock('../../../promotions/api')

const batch: CloseToExpiryBatch = {
  batch_id: 'b-1', lot_number: 'L1', expiry_date: '2026-10-30', days_left: 18, quantity: 7,
  warehouse_id: 'wh-1', warehouse_name: 'Amsterdam', sku_id: 'sku-1', sku_code: 'PQ-1',
  product_id: 'p-1', product_name: 'Pão de queijo', image_url: '', price_minor_units: 650, currency_code: 'EUR',
}

function setup() {
  const onClose = vi.fn()
  const onCreated = vi.fn()
  render(<ClearancePromotionDialog token="tok" batch={batch} onClose={onClose} onCreated={onCreated} />)
  return { onClose, onCreated }
}

beforeEach(() => {
  vi.mocked(promotionsApi.createPromotion).mockReset().mockResolvedValue({
    promotion: { id: 'promo-1', name: 'x' } as never, warnings: [],
  })
})

describe('ClearancePromotionDialog', () => {
  it('starts a fixed-price sale for the batch, ending on its expiry date', async () => {
    const { onCreated } = setup()
    fireEvent.change(screen.getByLabelText(/sale price/i), { target: { value: '4,99' } })
    expect(screen.getByText(/€4\.99/)).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: /start sale/i }))

    await waitFor(() => expect(onCreated).toHaveBeenCalledWith({ promotionId: 'promo-1', name: 'Clearance Pão de queijo (lot L1)', warnings: [] }))
    const input = vi.mocked(promotionsApi.createPromotion).mock.calls[0][1]
    expect(input).toMatchObject({
      name: 'Clearance Pão de queijo (lot L1)', type: 'fixed_price', status: 'active',
      lines: [{ sku_id: 'sku-1', batch_id: 'b-1', discount_type: 'fixed_price', discount_value: 499 }],
    })
    // Runs through the whole expiry day (local zone without a tenant time zone).
    expect(input.ends_at).toBe(zonedMidnight('2026-10-31', undefined).toISOString())
  })

  it('takes a percentage off', async () => {
    setup()
    fireEvent.click(screen.getByRole('radio', { name: /% off/i }))
    fireEvent.change(screen.getByLabelText(/discount/i), { target: { value: '30' } })
    fireEvent.click(screen.getByRole('button', { name: /start sale/i }))
    await waitFor(() => expect(promotionsApi.createPromotion).toHaveBeenCalled())
    expect(vi.mocked(promotionsApi.createPromotion).mock.calls[0][1].lines[0]).toMatchObject({ discount_type: 'percentage', discount_value: 3000 })
  })

  it('runs until the batch is gone when the end date is cleared', async () => {
    setup()
    fireEvent.change(screen.getByLabelText(/sale price/i), { target: { value: '5' } })
    fireEvent.change(screen.getByLabelText(/ends/i), { target: { value: '' } })
    fireEvent.click(screen.getByRole('button', { name: /start sale/i }))
    await waitFor(() => expect(promotionsApi.createPromotion).toHaveBeenCalled())
    expect(vi.mocked(promotionsApi.createPromotion).mock.calls[0][1].ends_at).toBeNull()
  })

  it('does not submit an invalid price', () => {
    setup()
    fireEvent.change(screen.getByLabelText(/sale price/i), { target: { value: '0' } })
    fireEvent.click(screen.getByRole('button', { name: /start sale/i }))
    expect(screen.getByRole('alert')).toHaveTextContent(/above zero/i)
    expect(promotionsApi.createPromotion).not.toHaveBeenCalled()
  })

  it('shows the API error and stays open', async () => {
    vi.mocked(promotionsApi.createPromotion).mockRejectedValue(new CodedApiError('PROMOTION_NAME_REQUIRED', 'name required'))
    const { onCreated } = setup()
    fireEvent.change(screen.getByLabelText(/sale price/i), { target: { value: '5' } })
    fireEvent.click(screen.getByRole('button', { name: /start sale/i }))
    await waitFor(() => expect(screen.getByRole('alert')).toBeInTheDocument())
    expect(onCreated).not.toHaveBeenCalled()
  })

  it('cancel closes without creating anything', () => {
    const { onClose } = setup()
    fireEvent.click(screen.getByRole('button', { name: /^cancel$/i }))
    expect(onClose).toHaveBeenCalled()
    expect(promotionsApi.createPromotion).not.toHaveBeenCalled()
  })
})
