// Calendar-day maths in a given IANA zone (the tenant's timezone; undefined
// means the browser's own zone, matching the date formatters). A calendar
// day is a "yyyy-mm-dd" string -- the same shape <input type="date">
// produces -- never a Date, so it can't drift a day across zones.

const DAY = /^(\d{4})-(\d{2})-(\d{2})$/

const formatters = new Map<string, Intl.DateTimeFormat>()

function zoneFormat(timeZone: string | undefined): Intl.DateTimeFormat {
  const key = timeZone ?? ''
  let f = formatters.get(key)
  if (!f) {
    const options: Intl.DateTimeFormatOptions = {
      year: 'numeric', month: '2-digit', day: '2-digit',
      hour: '2-digit', minute: '2-digit', second: '2-digit', hourCycle: 'h23',
    }
    try {
      f = new Intl.DateTimeFormat('en-GB', { ...options, timeZone })
    } catch {
      // Unknown IANA zone: use the browser's zone rather than break the page.
      f = new Intl.DateTimeFormat('en-GB', options)
    }
    formatters.set(key, f)
  }
  return f
}

// The instant's wall-clock fields in the zone, reinterpreted as if UTC.
function wallClockAsUTC(instant: number, timeZone: string | undefined): number {
  const p: Record<string, number> = {}
  for (const part of zoneFormat(timeZone).formatToParts(new Date(instant))) {
    if (part.type !== 'literal') p[part.type] = Number(part.value)
  }
  return Date.UTC(p.year, p.month - 1, p.day, p.hour === 24 ? 0 : p.hour, p.minute, p.second)
}

function offsetMs(instant: number, timeZone: string | undefined): number {
  return wallClockAsUTC(instant, timeZone) - (instant - (((instant % 1000) + 1000) % 1000))
}

function parseDay(day: string): [number, number, number] {
  const m = DAY.exec(day)
  if (!m) throw new Error(`calendarDay: not a yyyy-mm-dd day: ${day}`)
  return [Number(m[1]), Number(m[2]), Number(m[3])]
}

function formatDay(utcMs: number): string {
  return new Date(utcMs).toISOString().slice(0, 10)
}

// calendarDayInZone is the day the instant falls on in the zone.
export function calendarDayInZone(instant: Date, timeZone: string | undefined): string {
  return formatDay(wallClockAsUTC(instant.getTime(), timeZone))
}

// zonedMidnight is the instant 00:00 local time starts the day in the zone.
// The offset is re-read at the first guess so a DST change between UTC
// midnight and local midnight is honoured.
export function zonedMidnight(day: string, timeZone: string | undefined): Date {
  const [y, m, d] = parseDay(day)
  const utcMidnight = Date.UTC(y, m - 1, d)
  const guess = utcMidnight - offsetMs(utcMidnight, timeZone)
  return new Date(utcMidnight - offsetMs(guess, timeZone))
}

// addDays moves a calendar day by n days (negative goes back).
export function addDays(day: string, n: number): string {
  const [y, m, d] = parseDay(day)
  return formatDay(Date.UTC(y, m - 1, d + n))
}
