import type { RefObject } from 'react'
import { ScanBarcode } from 'lucide-react'

interface PackScanInputProps {
  inputRef: RefObject<HTMLInputElement>
  code: string
  onChange: (code: string) => void
  onSubmit: () => void
  disabled: boolean
  error: string
}

// The always-focused scan field; Enter submits the code.
export function PackScanInput({ inputRef, code, onChange, onSubmit, disabled, error }: PackScanInputProps) {
  return (
    <div className="px-6 py-4 border-b border-line shrink-0 dark:bg-night-900/80 bg-gray-50">
      <div className="flex-1 flex items-center gap-3 px-4 py-3 rounded-xl border-2  border-accent bg-raised">
        <ScanBarcode size={20} className="text-accent" />
        <input
          ref={inputRef}
          type="text"
          value={code}
          onChange={e => onChange(e.target.value)}
          onKeyDown={e => e.key === 'Enter' && onSubmit()}
          placeholder="Scan barcode or type and press Enter..."
          disabled={disabled}
          className="flex-1 bg-transparent text-sm text-ink outline-none focus-visible:ring-2 focus-visible:ring-focus/60 placeholder-ink-4 font-mono disabled:opacity-50"
        />
      </div>
      {error && <p className="text-2xs text-danger mt-2">{error}</p>}
    </div>
  )
}
