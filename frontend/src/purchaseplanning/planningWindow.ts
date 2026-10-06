// Bounds and defaults of the planning window -- same as the
// GET /procurement/recommendations contract.
export interface DayRange {
  min: number
  max: number
  fallback: number
  // Offered in the picker; a valid value from the URL outside this list is added.
  options: readonly number[]
}

export const COVERAGE: DayRange = { min: 1, max: 365, fallback: 30, options: [7, 14, 30, 60, 90, 180] }
export const SAFETY: DayRange = { min: 0, max: 90, fallback: 7, options: [0, 3, 7, 14, 30] }

// A day count from the URL; anything invalid (stale bookmark, hand-edited
// URL) reads as the default rather than failing the request.
export function parseDays(raw: string | null, range: DayRange): number {
  if (raw === null || !/^\d+$/.test(raw)) return range.fallback
  const n = Number(raw)
  return n >= range.min && n <= range.max ? n : range.fallback
}

export function dayOptions(range: DayRange, current: number): number[] {
  return [...new Set([...range.options, current])].sort((a, b) => a - b)
}
