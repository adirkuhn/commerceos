import { describe, it, expect } from 'vitest'
import i18next from 'i18next'
import { activityText } from '../activityText'

const t = (lng: string) => i18next.getFixedT(lng, 'orders')

describe('activityText', () => {
  it('names every known activity in the reader’s language', () => {
    const nl = t('nl')
    for (const type of ['marked_picked', 'myparcel_shipment_created', 'myparcel_shipment_failed', 'myparcel_shipment_cancelled',
      'myparcel_order_exported', 'myparcel_order_export_failed', 'myparcel_error']) {
      const text = activityText(nl, { activity_type: type, summary: 'English summary' })
      expect(text, type).not.toBe('English summary')
      expect(text, type).not.toMatch(/^activity\./)
    }
    expect(activityText(t('en'), { activity_type: 'marked_picked', summary: 'x' })).toBe('Marked as picked')
  })

  it('shows the stored summary for an unknown activity', () => {
    expect(activityText(t('nl'), { activity_type: 'something_new', summary: 'Stored' })).toBe('Stored')
  })
})
