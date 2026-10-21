import type { ShippingConnectionSettings } from '../types'

export type ShippingSettingsInput = Omit<ShippingConnectionSettings, 'connection_id'>

// The label defaults a carrier settings page edits; the rest of the
// connection's settings are kept as they are.
export type LabelDefaults = Pick<ShippingConnectionSettings,
  'default_label_format' | 'default_package_type' | 'default_signature_required' |
  'default_only_recipient' | 'insure_by_default' | 'auto_print_enabled'>

// What a connection without saved settings uses (backend defaults).
export const BUILT_IN_SETTINGS: ShippingSettingsInput = {
  default_carrier: 1, default_package_type: 1, default_label_format: 'pdf', default_label_size: 'A4',
  default_signature_required: false, default_only_recipient: false, insure_by_default: false,
  myparcel_account_id: null, myparcel_shop_id: null, auto_print_enabled: false, export_order_to_myparcel: false,
}

function withoutConnectionId(settings: ShippingConnectionSettings | null): ShippingSettingsInput {
  if (!settings) return BUILT_IN_SETTINGS
  const { connection_id: _connectionId, ...rest } = settings
  return { ...BUILT_IN_SETTINGS, ...rest }
}

// labelDefaultsOf reads the shown fields; a saved package type the
// provider doesn't offer becomes its first offered one.
export function labelDefaultsOf(settings: ShippingConnectionSettings | null, packageTypeIds: readonly number[]): LabelDefaults {
  const s = withoutConnectionId(settings)
  return {
    default_label_format: s.default_label_format,
    default_package_type: packageTypeIds.includes(s.default_package_type) ? s.default_package_type : packageTypeIds[0] ?? s.default_package_type,
    default_signature_required: s.default_signature_required,
    default_only_recipient: s.default_only_recipient,
    insure_by_default: s.insure_by_default,
    auto_print_enabled: s.auto_print_enabled,
  }
}

// settingsWithLabelDefaults is the full PUT body: the saved settings (or
// the built-in ones) with the shown fields replaced.
export function settingsWithLabelDefaults(settings: ShippingConnectionSettings | null, defaults: LabelDefaults): ShippingSettingsInput {
  return { ...withoutConnectionId(settings), ...defaults }
}
