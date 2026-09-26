import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from 'react'
import * as settingsApi from '../settings/api'
import type { TenantSettings } from '../settings/types'
import {
  DEFAULT_DATE_FORMAT, formatDate, formatDateTime, formatMonthDay, formatMonthYear, formatTime, parseDateFormat,
  type DateFormatSettings, type DateInput,
} from './date'

// Binds the date formatters to the tenant's Settings → Localization date
// format and timezone, loaded once per session. Until they load (or if
// loading fails) dates render in DEFAULT_DATE_FORMAT rather than blocking
// the page; saving Localization applies the new values right away via
// useApplyDateFormatSettings.

export interface DateFormatter {
  settings: DateFormatSettings
  formatDate: (value: DateInput) => string
  formatDateTime: (value: DateInput) => string
  formatTime: (value: DateInput) => string
  formatMonthDay: (value: DateInput) => string
  formatMonthYear: (value: DateInput, options?: { shortYear?: boolean }) => string
}

export function bindDateFormat(settings: DateFormatSettings): DateFormatter {
  return {
    settings,
    formatDate: value => formatDate(value, settings),
    formatDateTime: value => formatDateTime(value, settings),
    formatTime: value => formatTime(value, settings),
    formatMonthDay: value => formatMonthDay(value, settings),
    formatMonthYear: (value, options) => formatMonthYear(value, settings, options),
  }
}

export function toDateFormatSettings(s: Pick<TenantSettings, 'date_format' | 'timezone'>): DateFormatSettings {
  return { pattern: parseDateFormat(s.date_format), timeZone: s.timezone || undefined }
}

const defaultFormatter = bindDateFormat(DEFAULT_DATE_FORMAT)
const DateFormatContext = createContext<DateFormatter>(defaultFormatter)
const ApplyContext = createContext<(s: TenantSettings) => void>(() => {})

export function DateFormatProvider({ token, children }: { token: string; children: ReactNode }) {
  const [settings, setSettings] = useState<DateFormatSettings>(DEFAULT_DATE_FORMAT)

  useEffect(() => {
    if (!token) return
    let cancelled = false
    settingsApi.getTenantSettings(token)
      .then(s => { if (!cancelled) setSettings(toDateFormatSettings(s)) })
      .catch((e: unknown) => console.warn('date format: tenant settings unavailable, using the default', e))
    return () => { cancelled = true }
  }, [token])

  const apply = useCallback((s: TenantSettings) => setSettings(toDateFormatSettings(s)), [])
  const formatter = useMemo(() => bindDateFormat(settings), [settings])

  return (
    <ApplyContext.Provider value={apply}>
      <DateFormatContext.Provider value={formatter}>{children}</DateFormatContext.Provider>
    </ApplyContext.Provider>
  )
}

export function useDateFormat(): DateFormatter {
  return useContext(DateFormatContext)
}

export function useApplyDateFormatSettings(): (s: TenantSettings) => void {
  return useContext(ApplyContext)
}
