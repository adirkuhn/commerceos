import { clsx, type ClassValue } from 'clsx'
import { twMerge } from 'tailwind-merge'

export function cn(...inputs: ClassValue[]): string {
  return twMerge(clsx(inputs))
}

// formatMoney renders integer minor units (cents) using Intl -- every
// price in this app's types/data is stored as an integer to avoid float
// rounding errors, matching how a real backend would store money.
export function formatMoney(minorUnits: number, currency = 'USD', locale = 'en-US'): string {
  return new Intl.NumberFormat(locale, { style: 'currency', currency }).format(minorUnits / 100)
}

export function formatDate(iso: string, locale = 'en-US'): string {
  return new Intl.DateTimeFormat(locale, { year: 'numeric', month: 'short', day: 'numeric' }).format(new Date(iso))
}

export function slugify(value: string): string {
  return value
    .toLowerCase()
    .trim()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/(^-|-$)/g, '')
}

export function wait(ms: number): Promise<void> {
  return new Promise(resolve => setTimeout(resolve, ms))
}

// levenshtein powers search's fuzzy "did you mean" suggestion -- cheap
// enough for the ~30-item demo catalog; a real search backend
// (Elasticsearch/Meilisearch/Typesense/Algolia) would replace this
// entirely, see repositories/mock/MockSearchRepository.ts.
export function levenshtein(a: string, b: string): number {
  const m = a.length
  const n = b.length
  if (m === 0) return n
  if (n === 0) return m
  const dp: number[] = Array.from({ length: n + 1 }, (_, i) => i)
  for (let i = 1; i <= m; i++) {
    let prev = dp[0]
    dp[0] = i
    for (let j = 1; j <= n; j++) {
      const temp = dp[j]
      dp[j] = a[i - 1] === b[j - 1] ? prev : 1 + Math.min(prev, dp[j], dp[j - 1])
      prev = temp
    }
  }
  return dp[n]
}

export function highlightMatch(text: string, query: string): { before: string; match: string; after: string } | null {
  if (!query.trim()) return null
  const index = text.toLowerCase().indexOf(query.toLowerCase())
  if (index === -1) return null
  return { before: text.slice(0, index), match: text.slice(index, index + query.length), after: text.slice(index + query.length) }
}
