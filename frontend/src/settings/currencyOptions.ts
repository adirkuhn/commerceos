// Shared with SalesChannels.tsx's own currency picker (an internal
// channel's own currency, settable at creation -- see
// SalesChannelPayload.currencyCode's doc comment) so both pickers offer
// the exact same real-world currency list as the tenant's own default
// currency setting, not two independently-maintained lists that can
// silently drift apart.
export const CURRENCY_OPTIONS = [
  { value: 'EUR', label: 'EUR — Euro' },
  { value: 'USD', label: 'USD — US Dollar' },
  { value: 'GBP', label: 'GBP — British Pound' },
  { value: 'CHF', label: 'CHF — Swiss Franc' },
]
