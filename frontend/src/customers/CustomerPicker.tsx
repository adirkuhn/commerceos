import { useEffect, useState } from 'react'
import { UserRound, X } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { apiErrorText } from '../api/apiErrorText'
import { globalSearch } from '../search/api'
import type { SearchCustomerResult } from '../search/types'

export interface PickedCustomer {
  id: string
  name: string
  email: string
}

const MIN_CHARS = 2

function toPicked(c: SearchCustomerResult): PickedCustomer {
  const name = `${c.first_name} ${c.last_name}`.trim()
  return { id: c.id, name: name || c.email || c.id, email: c.email ?? '' }
}

// Picks an existing customer by name or email (global search's customers),
// or shows the chosen one with a remove button.
export function CustomerPicker({ token, value, onChange }: {
  token: string
  value: PickedCustomer | null
  onChange: (customer: PickedCustomer | null) => void
}) {
  const { t } = useTranslation('customers')
  const [input, setInput] = useState('')
  const [results, setResults] = useState<PickedCustomer[]>([])
  const [error, setError] = useState('')

  useEffect(() => {
    const query = input.trim()
    if (query.length < MIN_CHARS) {
      setResults([])
      return
    }
    let stale = false
    const timeout = setTimeout(async () => {
      try {
        const found = await globalSearch(token, query)
        if (!stale) {
          setResults(found.customers.map(toPicked))
          setError('')
        }
      } catch (e) {
        if (!stale) setError(apiErrorText(e) ?? t('picker.searchFailed'))
      }
    }, 250)
    return () => {
      stale = true
      clearTimeout(timeout)
    }
  }, [input, token, t])

  if (value) {
    return (
      <span className="inline-flex items-center gap-1.5 px-2 py-1 rounded-lg bg-sunken border border-line text-xs text-ink">
        <UserRound size={12} className="text-ink-4" />
        <span title={value.email}>{value.name}</span>
        <button onClick={() => onChange(null)} aria-label={t('picker.remove')} title={t('picker.remove')}
          className="w-4 h-4 flex items-center justify-center rounded hover:bg-hover text-ink-4"><X size={11} /></button>
      </span>
    )
  }

  return (
    <div className="relative w-64">
      <input
        value={input}
        onChange={e => setInput(e.target.value)}
        placeholder={t('picker.placeholder')}
        aria-label={t('picker.label')}
        className="w-full py-1.5 px-2.5 text-xs rounded-lg bg-raised border border-line text-ink placeholder-ink-4 outline-none focus:ring-1 focus:ring-focus"
      />
      {error && <p role="alert" className="mt-1 text-2xs text-danger">{error}</p>}
      {results.length > 0 && (
        <ul className="absolute z-20 mt-1 w-full rounded-lg border border-line bg-raised shadow-lg divide-y divide-line/60 max-h-60 overflow-auto">
          {results.map(c => (
            <li key={c.id}>
              <button
                onClick={() => { onChange(c); setInput(''); setResults([]) }}
                className="w-full text-left px-3 py-2 text-xs hover:bg-hover"
              >
                <span className="text-ink">{c.name}</span>
                {c.email && c.email !== c.name && <span className="block text-2xs text-ink-4">{c.email}</span>}
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
