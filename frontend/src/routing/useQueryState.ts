import { useCallback } from 'react'
import { useSearchParams } from 'react-router-dom'

// URL-backed replacements for useState, for view state that must survive a
// refresh and work with Back: tabs, sub-views, pagination, selected codes.
// Throwaway UI state (form inputs, confirm dialogs, busy flags) stays in
// useState -- only "where am I" belongs in the URL.

export type HistoryMode = 'push' | 'replace'

export interface QueryStateOptions {
  // 'replace' (default) for modifiers like tabs/pages so every click doesn't
  // add a Back step; 'push' when changing the value is a real navigation
  // the user expects Back to undo.
  history?: HistoryMode
  // Params cleared in the same write whenever this one changes -- e.g. a
  // search box resets "page". Two separate setters in one handler would
  // each start from the same URL and the second would undo the first.
  resets?: readonly string[]
}

type Setter<T> = (value: T) => void

// Writes one param while preserving every other param on the URL. A null
// value removes the param, which is how defaults stay out of the URL.
function useParamWriter(key: string, options: QueryStateOptions): Setter<string | null> {
  const [, setSearchParams] = useSearchParams()
  const history = options.history ?? 'replace'
  const resets = (options.resets ?? []).join(',')
  return useCallback((value: string | null) => {
    setSearchParams(prev => {
      const next = new URLSearchParams(prev)
      if (value === null) next.delete(key)
      else next.set(key, value)
      if (resets) for (const r of resets.split(',')) next.delete(r)
      return next
    }, { replace: history === 'replace' })
  }, [key, history, resets, setSearchParams])
}

export function useQueryString(key: string, options: QueryStateOptions = {}): [string | null, Setter<string | null>] {
  const [searchParams] = useSearchParams()
  const write = useParamWriter(key, options)
  const set = useCallback((value: string | null) => write(value === '' ? null : value), [write])
  return [searchParams.get(key), set]
}

// A param restricted to a fixed set of values (tabs, view modes). Anything
// outside the set -- stale bookmark, hand-edited URL -- reads as the default
// rather than rendering a page state that doesn't exist.
export function useQueryEnum<T extends string>(
  key: string,
  allowed: readonly T[],
  defaultValue: T,
  options: QueryStateOptions = {},
): [T, Setter<T>] {
  const [searchParams] = useSearchParams()
  const write = useParamWriter(key, options)
  const raw = searchParams.get(key)
  const value = raw !== null && (allowed as readonly string[]).includes(raw) ? (raw as T) : defaultValue
  const set = useCallback((next: T) => write(next === defaultValue ? null : next), [write, defaultValue])
  return [value, set]
}

export function useQueryPage(key = 'page', options: QueryStateOptions = {}): [number, Setter<number>] {
  const [searchParams] = useSearchParams()
  const write = useParamWriter(key, options)
  const raw = searchParams.get(key)
  const parsed = raw !== null && /^\d+$/.test(raw) ? Number(raw) : 1
  const value = parsed >= 1 ? parsed : 1
  const set = useCallback((next: number) => write(next <= 1 ? null : String(next)), [write])
  return [value, set]
}
