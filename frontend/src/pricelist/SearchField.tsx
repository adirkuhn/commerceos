import { useRef } from 'react'
import { Search, X } from 'lucide-react'
import { useTranslation } from 'react-i18next'

// A search box with a clear (×) button; Escape clears too.
export function SearchField({ label, value, onChange }: { label: string; value: string; onChange: (value: string) => void }) {
  const { t } = useTranslation('priceList')
  const input = useRef<HTMLInputElement>(null)

  function clear() {
    onChange('')
    input.current?.focus()
  }

  return (
    <div className="relative">
      <Search size={13} className="absolute left-3 top-1/2 -translate-y-1/2 text-ink-4" />
      <input
        ref={input}
        value={value}
        onChange={e => onChange(e.target.value)}
        onKeyDown={e => { if (e.key === 'Escape' && value) clear() }}
        placeholder={label}
        aria-label={label}
        className="w-full pl-8 pr-8 py-1.5 text-xs rounded-lg bg-raised border-line border text-ink placeholder-ink-4 outline-none focus:ring-1 focus:ring-focus"
      />
      {value && (
        <button type="button" onClick={clear} aria-label={t('clearSearch')} title={t('clearSearch')}
          className="absolute right-1.5 top-1/2 -translate-y-1/2 w-5 h-5 flex items-center justify-center rounded hover:bg-hover text-ink-4 hover:text-ink-2">
          <X size={12} />
        </button>
      )}
    </div>
  )
}
