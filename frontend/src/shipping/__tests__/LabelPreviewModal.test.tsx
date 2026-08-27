// @vitest-environment jsdom
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi } from 'vitest'
import { LabelPreviewModal } from '../LabelPreviewModal'
import type { RecipientAddress } from '../types'

const recipient: RecipientAddress = {
  person: 'Jane Doe', street: 'Damrak', number: '1', number_suffix: 'A',
  postal_code: '1012AB', city: 'Amsterdam', cc: 'NL',
}

describe('LabelPreviewModal', () => {
  it('shows the confirmed data laid out label-style, with carrier and size', () => {
    render(<LabelPreviewModal recipient={recipient} carrier={1} labelSize="A4" onLabelSizeChange={vi.fn()} onBack={vi.fn()} onPrint={vi.fn()} onClose={vi.fn()} />)

    expect(screen.getByText('Jane Doe')).toBeInTheDocument()
    expect(screen.getByText('Damrak 1A')).toBeInTheDocument()
    expect(screen.getByText('1012AB Amsterdam')).toBeInTheDocument()
    expect(screen.getByText('NL')).toBeInTheDocument()
    expect(screen.getByText('PostNL')).toBeInTheDocument()
    // "A4" appears twice -- the preview chip and the size selector button.
    expect(screen.getAllByText('A4')).toHaveLength(2)
  })

  it('calls onLabelSizeChange when a different size is picked, without calling onPrint', async () => {
    const onLabelSizeChange = vi.fn()
    const onPrint = vi.fn()
    render(<LabelPreviewModal recipient={recipient} carrier={1} labelSize="A4" onLabelSizeChange={onLabelSizeChange} onBack={vi.fn()} onPrint={onPrint} onClose={vi.fn()} />)

    await userEvent.click(screen.getByRole('button', { name: 'A6' }))

    expect(onLabelSizeChange).toHaveBeenCalledWith('A6')
    expect(onPrint).not.toHaveBeenCalled()
  })

  it('calls onBack (not onPrint, not onClose) when "Back to edit" is clicked', async () => {
    const onBack = vi.fn()
    const onPrint = vi.fn()
    render(<LabelPreviewModal recipient={recipient} carrier={1} labelSize="A4" onLabelSizeChange={vi.fn()} onBack={onBack} onPrint={onPrint} onClose={vi.fn()} />)

    await userEvent.click(screen.getByRole('button', { name: 'Back to edit' }))

    expect(onBack).toHaveBeenCalled()
    expect(onPrint).not.toHaveBeenCalled()
  })

  it('calls onPrint when "Print label" is clicked -- this is what actually creates the shipment', async () => {
    const onPrint = vi.fn().mockResolvedValue(undefined)
    render(<LabelPreviewModal recipient={recipient} carrier={1} labelSize="A4" onLabelSizeChange={vi.fn()} onBack={vi.fn()} onPrint={onPrint} onClose={vi.fn()} />)

    await userEvent.click(screen.getByRole('button', { name: 'Print label' }))

    expect(onPrint).toHaveBeenCalled()
  })

  it('shows an error and stays open when onPrint rejects (MyParcel rejected the shipment)', async () => {
    const onPrint = vi.fn().mockRejectedValue(new Error('invalid postal_code'))
    render(<LabelPreviewModal recipient={recipient} carrier={1} labelSize="A4" onLabelSizeChange={vi.fn()} onBack={vi.fn()} onPrint={onPrint} onClose={vi.fn()} />)

    await userEvent.click(screen.getByRole('button', { name: 'Print label' }))

    expect(await screen.findByText('invalid postal_code')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Print label' })).not.toBeDisabled()
  })

  it('calls onClose when the close button is clicked', async () => {
    const onClose = vi.fn()
    render(<LabelPreviewModal recipient={recipient} carrier={1} labelSize="A4" onLabelSizeChange={vi.fn()} onBack={vi.fn()} onPrint={vi.fn()} onClose={onClose} />)

    await userEvent.click(screen.getByLabelText('Close'))

    expect(onClose).toHaveBeenCalled()
  })
})
