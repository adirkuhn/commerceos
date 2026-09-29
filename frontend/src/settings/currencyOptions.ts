import i18next from 'i18next'

// Shared with SalesChannels.tsx's own currency picker (an internal
// channel's own currency, settable at creation -- see
// SalesChannelPayload.currencyCode's doc comment) so both pickers offer
// the exact same real-world currency list as the tenant's own default
// currency setting, not two independently-maintained lists that can
// silently drift apart. Names come from the browser in the admin language.
export const CURRENCY_CODES = ['EUR', 'USD', 'GBP', 'CHF'] as const

function currencyName(code: string): string {
  try {
    return new Intl.DisplayNames([i18next.language || 'en'], { type: 'currency' }).of(code) ?? code
  } catch {
    return code
  }
}

export function currencyOptions(): { value: string; label: string }[] {
  return CURRENCY_CODES.map(code => ({ value: code, label: `${code} — ${currencyName(code)}` }))
}
