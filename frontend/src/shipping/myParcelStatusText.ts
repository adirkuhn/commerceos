import type { TFunction } from 'i18next'

// MyParcel's published shipment status codes (backend
// domain/shipping.myParcelStatusLabels).
export const MYPARCEL_STATUS_CODES = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 30, 31, 32, 33, 34, 35, 36, 37, 38] as const

// myParcelStatusText is a MyParcel tracking status in the reader's language.
// Keys are looked up by code at runtime (kept by i18next-parser.config's
// keepRemoved); an unknown code names its number.
export function myParcelStatusText(t: TFunction<'shipping'>, code: number, fallback: string): string {
  if ((MYPARCEL_STATUS_CODES as readonly number[]).includes(code)) return t(`myparcelStatus.s${code}` as 'myparcelStatus.s1')
  return t('myparcelStatus.unknown', { code, defaultValue: fallback })
}
