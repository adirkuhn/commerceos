import { describe, it, expect } from 'vitest'
import i18next from 'i18next'
import { cancelOutcomeText, outcomeOfCancelResult, outcomeOfShipment } from '../cancelOutcomeText'

const t = (lng: string) => i18next.getFixedT(lng, 'shipping')

describe('cancelOutcomeText', () => {
  it('says MyParcel deleted it, in the reader’s language', () => {
    expect(cancelOutcomeText(t('en'), { deleted: true, carrierError: '', labelKept: false }, 'x')).toBe('Shipment deleted on MyParcel.')
    expect(cancelOutcomeText(t('nl'), { deleted: true, carrierError: '', labelKept: false }, 'x')).toBe('Zending verwijderd bij MyParcel.')
  })

  it('includes MyParcel’s own reason when it refused', () => {
    const text = cancelOutcomeText(t('pt-BR'), { deleted: false, carrierError: '422 label printed', labelKept: false }, 'x')
    expect(text).toContain('422 label printed')
    expect(text).toContain('MyParcel')
    expect(text).not.toContain('could not delete')
  })

  it('falls back to the stored English note without an outcome', () => {
    expect(cancelOutcomeText(t('nl'), null, 'Stored note')).toBe('Stored note')
  })

  it('explains a kept label (PostNL) without calling it a refusal, in every language', () => {
    const kept = { deleted: false, carrierError: '', labelKept: true }
    expect(cancelOutcomeText(t('en'), kept, 'x')).toContain('7 days')
    expect(cancelOutcomeText(t('en'), kept, 'x')).not.toContain('could not delete')
    expect(cancelOutcomeText(t('nl'), kept, 'x')).toContain('7 dagen')
    expect(cancelOutcomeText(t('pt-BR'), kept, 'x')).toContain('7 dias')
  })
})

describe('outcomeOfShipment', () => {
  it('reads a stored deleted / refused outcome', () => {
    expect(outcomeOfShipment({ cancel_deleted_on_carrier: true })).toEqual({ deleted: true, carrierError: '', labelKept: false })
    expect(outcomeOfShipment({ cancel_deleted_on_carrier: false, cancel_carrier_error: 'nope' })).toEqual({ deleted: false, carrierError: 'nope', labelKept: false })
  })

  it('reads a kept label', () => {
    expect(outcomeOfShipment({ cancel_deleted_on_carrier: false, cancel_label_kept: true })).toEqual({ deleted: false, carrierError: '', labelKept: true })
  })

  it('has no outcome for shipments cancelled before outcomes were stored', () => {
    expect(outcomeOfShipment({ cancel_deleted_on_carrier: null })).toBeNull()
    expect(outcomeOfShipment({})).toBeNull()
  })
})

describe('outcomeOfCancelResult', () => {
  it('maps the cancel call result', () => {
    expect(outcomeOfCancelResult({ deleted_on_myparcel: true, message: 'm' })).toEqual({ deleted: true, carrierError: '', labelKept: false })
    expect(outcomeOfCancelResult({ deleted_on_myparcel: false, carrier_error: 'e', message: 'm' })).toEqual({ deleted: false, carrierError: 'e', labelKept: false })
    expect(outcomeOfCancelResult({ deleted_on_myparcel: false, label_kept: true, message: 'm' })).toEqual({ deleted: false, carrierError: '', labelKept: true })
  })
})
