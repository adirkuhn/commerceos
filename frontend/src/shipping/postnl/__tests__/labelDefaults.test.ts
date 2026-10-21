import { describe, it, expect } from 'vitest'
import { BUILT_IN_SETTINGS, labelDefaultsOf, settingsWithLabelDefaults } from '../labelDefaults'
import type { ShippingConnectionSettings } from '../../types'

const saved: ShippingConnectionSettings = {
  connection_id: 'pn', default_carrier: 1, default_package_type: 2, default_label_format: 'zpl', default_label_size: 'A6',
  default_signature_required: true, default_only_recipient: false, insure_by_default: true,
  myparcel_account_id: 7, myparcel_shop_id: 8, auto_print_enabled: true, export_order_to_myparcel: false,
}
const POSTNL_TYPES = [1, 2, 6, 7]

describe('labelDefaultsOf', () => {
  it('reads the shown fields from saved settings', () => {
    expect(labelDefaultsOf(saved, POSTNL_TYPES)).toEqual({
      default_label_format: 'zpl', default_package_type: 2, default_signature_required: true,
      default_only_recipient: false, insure_by_default: true, auto_print_enabled: true,
    })
  })

  it('uses the built-in defaults without saved settings', () => {
    expect(labelDefaultsOf(null, POSTNL_TYPES)).toMatchObject({ default_label_format: 'pdf', default_package_type: 1, auto_print_enabled: false })
  })

  it('falls back to the first allowed package type when the saved one is not offered', () => {
    expect(labelDefaultsOf({ ...saved, default_package_type: 5 }, POSTNL_TYPES).default_package_type).toBe(1)
  })
})

describe('settingsWithLabelDefaults', () => {
  it('keeps every field the page does not show and drops the connection id', () => {
    const update = settingsWithLabelDefaults(saved, { ...labelDefaultsOf(saved, POSTNL_TYPES), default_label_format: 'pdf', auto_print_enabled: false })
    expect(update).toEqual({
      default_carrier: 1, default_package_type: 2, default_label_format: 'pdf', default_label_size: 'A6',
      default_signature_required: true, default_only_recipient: false, insure_by_default: true,
      myparcel_account_id: 7, myparcel_shop_id: 8, auto_print_enabled: false, export_order_to_myparcel: false,
    })
  })

  it('starts from the built-in defaults without saved settings', () => {
    const update = settingsWithLabelDefaults(null, labelDefaultsOf(null, POSTNL_TYPES))
    expect(update).toEqual(BUILT_IN_SETTINGS)
  })
})
