import { STORE_LOCALE } from '../../core/money'

const fmt = new Intl.DateTimeFormat(STORE_LOCALE, { day: 'numeric', month: 'long', year: 'numeric', timeZone: 'UTC' })

/** Formats dates the same on the server and in the browser (fixed locale and zone). */
export function useDateLabel() {
  return (iso: string) => {
    const d = new Date(iso)
    return Number.isNaN(d.getTime()) ? '' : fmt.format(d)
  }
}
