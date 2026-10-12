import i18next from 'i18next'

const MISSING = '—'

// The tenant's Settings → Localization → Date format. Every date shown in
// the app goes through here (see useDateFormat for the tenant-bound
// versions) -- never through toLocaleDateString/toLocaleString, which
// follow the browser's locale instead of the tenant's setting.
// Pinned to the settings OpenAPI DateFormat enum (see enumContract).
export type DateFormat = 'DD/MM/YYYY' | 'MM/DD/YYYY' | 'YYYY-MM-DD'
export const DATE_FORMATS: readonly DateFormat[] = ['DD/MM/YYYY', 'MM/DD/YYYY', 'YYYY-MM-DD']

export interface DateFormatSettings {
  pattern: DateFormat
  // IANA zone timestamps are shown in (the tenant's timezone setting);
  // undefined means the browser's own zone.
  timeZone?: string
}

export const DEFAULT_DATE_FORMAT: DateFormatSettings = { pattern: 'DD/MM/YYYY' }

export type DateInput = string | Date | null | undefined

// An empty or unknown stored value (older tenants saved '') falls back to
// the default the Localization screen itself shows.
export function parseDateFormat(value: string | null | undefined): DateFormat {
  return DATE_FORMATS.find(p => p === value) ?? DEFAULT_DATE_FORMAT.pattern
}

interface Parts { year: string; month: string; day: string; hour: string; minute: string }

const DATE_ONLY = /^(\d{4})-(\d{2})-(\d{2})$/

// A date-only value (an expiry, a sale end date) is a calendar day, not an
// instant -- it's split as-is, never run through a timezone, or it would
// show a day early west of UTC.
function toParts(value: DateInput, timeZone: string | undefined): Parts | null {
  if (!value) return null
  if (typeof value === 'string') {
    const m = DATE_ONLY.exec(value)
    if (m) return { year: m[1], month: m[2], day: m[3], hour: '00', minute: '00' }
  }
  const date = value instanceof Date ? value : new Date(value)
  if (Number.isNaN(date.getTime())) return null
  const found: Record<string, string> = {}
  for (const p of dateTimeFormat(timeZone).formatToParts(date)) found[p.type] = p.value
  return {
    year: found.year, month: found.month, day: found.day,
    // Some engines render midnight as 24 with hourCycle h23 unset.
    hour: found.hour === '24' ? '00' : found.hour, minute: found.minute,
  }
}

const formatters = new Map<string, Intl.DateTimeFormat>()

function dateTimeFormat(timeZone: string | undefined): Intl.DateTimeFormat {
  const key = timeZone ?? ''
  let f = formatters.get(key)
  if (!f) {
    const options: Intl.DateTimeFormatOptions = {
      year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', hourCycle: 'h23',
    }
    try {
      f = new Intl.DateTimeFormat('en-GB', { ...options, timeZone })
    } catch {
      // Unknown IANA zone: show the browser's zone rather than break the page.
      f = new Intl.DateTimeFormat('en-GB', options)
    }
    formatters.set(key, f)
  }
  return f
}

function datePart(p: Parts, pattern: DateFormat): string {
  switch (pattern) {
    case 'MM/DD/YYYY': return `${p.month}/${p.day}/${p.year}`
    case 'YYYY-MM-DD': return `${p.year}-${p.month}-${p.day}`
    default: return `${p.day}/${p.month}/${p.year}`
  }
}

export function formatDate(value: DateInput, settings: DateFormatSettings = DEFAULT_DATE_FORMAT): string {
  const p = toParts(value, settings.timeZone)
  return p ? datePart(p, settings.pattern) : MISSING
}

export function formatDateTime(value: DateInput, settings: DateFormatSettings = DEFAULT_DATE_FORMAT): string {
  const p = toParts(value, settings.timeZone)
  return p ? `${datePart(p, settings.pattern)} ${p.hour}:${p.minute}` : MISSING
}

export function formatTime(value: DateInput, settings: DateFormatSettings = DEFAULT_DATE_FORMAT): string {
  const p = toParts(value, settings.timeZone)
  return p ? `${p.hour}:${p.minute}` : MISSING
}

// Short month names in the admin language (read when formatting, so a
// language switch applies at once).
function monthShort(month: number): string {
  switch (month) {
    case 1: return i18next.t('ui:monthShort.jan')
    case 2: return i18next.t('ui:monthShort.feb')
    case 3: return i18next.t('ui:monthShort.mar')
    case 4: return i18next.t('ui:monthShort.apr')
    case 5: return i18next.t('ui:monthShort.may')
    case 6: return i18next.t('ui:monthShort.jun')
    case 7: return i18next.t('ui:monthShort.jul')
    case 8: return i18next.t('ui:monthShort.aug')
    case 9: return i18next.t('ui:monthShort.sep')
    case 10: return i18next.t('ui:monthShort.oct')
    case 11: return i18next.t('ui:monthShort.nov')
    default: return i18next.t('ui:monthShort.dec')
  }
}

// Short labels for charts and report periods ("3 Sep", "Sep 2026"): day
// and month follow the tenant pattern's order (month first only for
// MM/DD/YYYY).
export function formatMonthDay(value: DateInput, settings: DateFormatSettings = DEFAULT_DATE_FORMAT): string {
  const p = toParts(value, settings.timeZone)
  if (!p) return MISSING
  const day = String(Number(p.day))
  const month = monthShort(Number(p.month))
  return settings.pattern === 'MM/DD/YYYY' ? `${month} ${day}` : `${day} ${month}`
}

export function formatMonthYear(
  value: DateInput, settings: DateFormatSettings = DEFAULT_DATE_FORMAT, { shortYear = false }: { shortYear?: boolean } = {},
): string {
  const p = toParts(value, settings.timeZone)
  if (!p) return MISSING
  return `${monthShort(Number(p.month))} ${shortYear ? `'${p.year.slice(2)}` : p.year}`
}

// A heading-style day ("Monday, October 12" / "maandag 12 oktober") in the
// admin language, in the tenant's timezone.
export function formatLongDay(value: Date, language: string, settings: DateFormatSettings = DEFAULT_DATE_FORMAT): string {
  const options: Intl.DateTimeFormatOptions = { weekday: 'long', day: 'numeric', month: 'long' }
  try {
    return new Intl.DateTimeFormat(language, { ...options, timeZone: settings.timeZone }).format(value)
  } catch {
    return new Intl.DateTimeFormat(language, options).format(value)
  }
}
