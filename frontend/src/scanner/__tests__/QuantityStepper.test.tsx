// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { QuantityStepper } from '../QuantityStepper'

function typeInto(value: string, props: Partial<Parameters<typeof QuantityStepper>[0]> = {}) {
  cleanup()
  const onChange = vi.fn()
  render(<QuantityStepper id="q" label="Qty" value="" onChange={onChange} {...props} />)
  fireEvent.change(screen.getByRole('textbox', { name: 'Qty' }), { target: { value } })
  return onChange
}

describe('QuantityStepper', () => {
  it('whole numbers only by default', () => {
    expect(typeInto('2.5')).toHaveBeenCalledWith('25')
  })

  describe('decimal (kg)', () => {
    it('keeps a decimal with a comma or dot', () => {
      expect(typeInto('2,5', { decimal: true })).toHaveBeenCalledWith('2,5')
      expect(typeInto('0.125', { decimal: true })).toHaveBeenCalledWith('0.125')
    })

    it('keeps a half-typed decimal', () => {
      expect(typeInto('1,', { decimal: true })).toHaveBeenCalledWith('1,')
    })

    it('ignores letters and a 4th decimal', () => {
      expect(typeInto('1a', { decimal: true })).not.toHaveBeenCalled()
      expect(typeInto('1.2345', { decimal: true })).not.toHaveBeenCalled()
    })

    it('clamps a complete number to max', () => {
      expect(typeInto('60', { decimal: true, max: 50 })).toHaveBeenCalledWith('50')
    })
  })
})
