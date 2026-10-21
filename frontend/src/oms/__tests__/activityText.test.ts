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

  it('names the shipping provider the label was made with', () => {
    const en = t('en')
    expect(activityText(en, { activity_type: 'myparcel_shipment_created', summary: 'x', metadata: { provider: 'postnl' } })).toBe('Shipment created via PostNL')
    expect(activityText(en, { activity_type: 'myparcel_shipment_failed', summary: 'x', metadata: { provider: 'postnl' } })).toBe('Failed to create PostNL shipment')
    expect(activityText(t('nl'), { activity_type: 'myparcel_shipment_created', summary: 'x', metadata: { provider: 'postnl' } })).toBe('Zending aangemaakt via PostNL')
  })

  it('older shipment activities without a provider were MyParcel', () => {
    expect(activityText(t('en'), { activity_type: 'myparcel_shipment_created', summary: 'x', metadata: {} })).toBe('Shipment created via MyParcel')
    expect(activityText(t('en'), { activity_type: 'myparcel_shipment_failed', summary: 'x' })).toBe('Failed to create MyParcel shipment')
  })

  it('shows the stored summary for an unknown activity', () => {
    expect(activityText(t('nl'), { activity_type: 'something_new', summary: 'Stored' })).toBe('Stored')
  })
})
