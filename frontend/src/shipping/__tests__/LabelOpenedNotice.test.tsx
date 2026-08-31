// @vitest-environment jsdom
import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { LabelOpenedNotice } from '../LabelOpenedNotice'

describe('LabelOpenedNotice', () => {
  it('calls onPrint when Print is clicked', () => {
    const onPrint = vi.fn()
    render(<LabelOpenedNotice onPrint={onPrint} onDismiss={vi.fn()} />)

    fireEvent.click(screen.getByRole('button', { name: /Print/ }))
    expect(onPrint).toHaveBeenCalled()
  })

  it('calls onDismiss when the close button is clicked', () => {
    const onDismiss = vi.fn()
    render(<LabelOpenedNotice onPrint={vi.fn()} onDismiss={onDismiss} />)

    fireEvent.click(screen.getByLabelText('Dismiss'))
    expect(onDismiss).toHaveBeenCalled()
  })
})
