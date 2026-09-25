const MISSING = '—'

// formatDate renders an ISO timestamp as a localized date; a missing value
// renders as an em dash.
export function formatDate(iso?: string | null): string {
  return iso ? new Date(iso).toLocaleDateString() : MISSING
}

// formatDateTime renders an ISO timestamp as a localized date and time; a
// missing value renders as an em dash.
export function formatDateTime(iso?: string | null): string {
  return iso ? new Date(iso).toLocaleString() : MISSING
}
