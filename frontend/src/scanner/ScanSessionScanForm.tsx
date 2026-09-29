import type { FormEvent, ReactNode, RefObject } from 'react'
import { Camera } from 'lucide-react'
import { useTranslation } from 'react-i18next'

interface ScanSessionScanFormProps {
  inputRef: RefObject<HTMLInputElement>
  code: string
  onCodeChange: (code: string) => void
  onSubmit: (e: FormEvent) => void
  onOpenCamera: () => void
  error: string
  scanning: boolean
  readOnly: boolean
  // A mis-scan alert is open -- scanning stays blocked until it's dismissed.
  blocked: boolean
  // Rendered between the error and the input (e.g. search matches).
  children?: ReactNode
}

// The scan/type-a-barcode box pinned to the bottom of a session screen.
export function ScanSessionScanForm({ inputRef, code, onCodeChange, onSubmit, onOpenCamera, error, scanning, readOnly, blocked, children }: ScanSessionScanFormProps) {
  const { t } = useTranslation('scanner')
  return (
    <form onSubmit={onSubmit} className="border-t border-night-700 p-4 space-y-2 shrink-0">
      {error && <p className="text-sm text-red-400">{error}</p>}
      {children}
      <div className="flex gap-2">
        <input
          ref={inputRef}
          value={code}
          onChange={e => onCodeChange(e.target.value)}
          placeholder={t('shared.scanPlaceholder')}
          disabled={scanning || readOnly || blocked}
          className="flex-1 px-3 py-3 text-sm rounded-xl bg-night-800 border border-night-700 text-gray-100 outline-none focus:ring-2 focus:ring-focus disabled:opacity-50"
        />
        <button aria-label={t('shared.useCamera')}
          type="button"
          onClick={onOpenCamera}
          title={t('shared.useCamera')}
          className="w-11 shrink-0 rounded-xl bg-night-800 border border-night-700 flex items-center justify-center text-gray-300"
        >
          <Camera size={16} />
        </button>
      </div>
      <button
        type="submit" disabled={scanning || !code.trim() || blocked}
        className="w-full py-3 rounded-xl bg-accent-solid text-white font-semibold hover:bg-accent-solid-hover disabled:opacity-50"
      >
        {scanning ? t('shared.lookingUp') : t('shared.scan')}
      </button>
    </form>
  )
}
