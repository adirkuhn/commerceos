import type { RefObject } from 'react'
import { ScanBarcode } from 'lucide-react'
import { useTranslation } from 'react-i18next'

interface PickScanInputProps {
  inputRef: RefObject<HTMLInputElement>
  code: string
  onCodeChange: (code: string) => void
  onSubmit: () => void
  disabled: boolean
  error: string
}

// The always-focused scan box: a scanner "types" the code and presses Enter.
export function PickScanInput({ inputRef, code, onCodeChange, onSubmit, disabled, error }: PickScanInputProps) {
  const { t } = useTranslation('picking')
  return (
    <div className="px-6 py-4 border-b border-line shrink-0 bg-sunken">
      <div className="flex-1 flex items-center gap-3 px-4 py-3 rounded-xl border-2  border-accent bg-raised">
        <ScanBarcode size={20} className="text-accent" />
        <input
          ref={inputRef}
          type="text"
          value={code}
          onChange={e => onCodeChange(e.target.value)}
          onKeyDown={e => e.key === 'Enter' && onSubmit()}
          placeholder={t('scanInput.placeholder')}
          disabled={disabled}
          className="flex-1 bg-transparent text-sm text-ink outline-none focus-visible:ring-2 focus-visible:ring-focus/60 placeholder-ink-4 font-mono disabled:opacity-50"
        />
      </div>
      {error && <p className="text-2xs text-danger mt-2">{error}</p>}
    </div>
  )
}
