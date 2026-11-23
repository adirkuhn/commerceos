// How long ago a chat was started, in the reader's language ("5 minutes
// ago", "yesterday"); a week or more shows the date instead.
export function relativeTime(iso: string, lang: string, now: Date = new Date(), formatDate: (iso: string) => string = d => d.slice(0, 10)): string {
  const seconds = Math.round((new Date(iso).getTime() - now.getTime()) / 1000)
  const abs = Math.abs(seconds)
  const rtf = new Intl.RelativeTimeFormat(lang, { numeric: 'auto' })
  if (abs < 45) return rtf.format(0, 'second')
  if (abs < 3600) return rtf.format(Math.round(seconds / 60), 'minute')
  if (abs < 86400) return rtf.format(Math.round(seconds / 3600), 'hour')
  if (abs < 7 * 86400) return rtf.format(Math.round(seconds / 86400), 'day')
  return formatDate(iso)
}
