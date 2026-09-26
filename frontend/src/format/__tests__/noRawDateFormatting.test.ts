import { describe, it, expect } from 'vitest'

// Every date shown in the app must follow the tenant's Settings →
// Localization date format and timezone, i.e. go through format/date (or
// useDateFormat). The browser's locale formatters silently ignore that
// setting -- this pins that nobody reaches for them again.
const sourceFiles = import.meta.glob(['../../**/*.ts', '../../**/*.tsx', '!../../**/*.test.*', '!../../**/__tests__/**', '!../../format/**'], {
  query: '?raw', import: 'default', eager: true,
}) as Record<string, string>
const RAW_DATE_FORMATTING = [
  /\.toLocaleDateString\(/,
  /\.toLocaleTimeString\(/,
  /(new Date\([^)]*\)|\b\w*(date|Date|_at|At|time|Time)\)?)\??\.toLocaleString\(/,
  /new Intl\.DateTimeFormat\(/,
]

describe('date formatting', () => {
  it('never formats a date with the browser locale outside src/format', () => {
    const files = Object.entries(sourceFiles)
    expect(files.length).toBeGreaterThan(100)
    const offenders = files.flatMap(([path, source]) => source.split('\n').flatMap((line, i) =>
      RAW_DATE_FORMATTING.some(re => re.test(line)) ? [`${path.replace('../../', '')}:${i + 1}: ${line.trim()}`] : []))

    expect(offenders).toEqual([])
  })
})
