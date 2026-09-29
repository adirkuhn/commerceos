import clsx from 'clsx'
import { useCallback, useEffect, useRef, useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { Dialog } from './Dialog'

export interface ConfirmOptions {
  // The question, e.g. 'Delete coupon "SPRING"?'
  title: string
  // What happens / what can't be undone.
  message?: string
  // Names the action ("Delete coupon"), never a generic OK.
  confirmLabel: string
  destructive?: boolean
}

interface Pending {
  options: ConfirmOptions
  resolve: (ok: boolean) => void
}

// In-app replacement for window.confirm: `await confirm({...})` resolves
// true/false, and the owning component renders `confirmDialog`. Kept local
// to the component (no provider) so each screen owns its own prompt.
export function useConfirm(): { confirm: (options: ConfirmOptions) => Promise<boolean>; confirmDialog: ReactNode } {
  const [pending, setPending] = useState<Pending | null>(null)
  const pendingRef = useRef<Pending | null>(null)
  pendingRef.current = pending

  useEffect(() => () => pendingRef.current?.resolve(false), [])

  const confirm = useCallback((options: ConfirmOptions) => new Promise<boolean>(resolve => {
    setPending({ options, resolve })
  }), [])

  const settle = (ok: boolean) => {
    pending?.resolve(ok)
    setPending(null)
  }

  const confirmDialog = pending ? (
    <ConfirmDialog options={pending.options} onCancel={() => settle(false)} onConfirm={() => settle(true)} />
  ) : null

  return { confirm, confirmDialog }
}

function ConfirmDialog({ options, onCancel, onConfirm }: { options: ConfirmOptions; onCancel: () => void; onConfirm: () => void }) {
  const { t } = useTranslation('common')
  return (
    // No default button (focus on the dialog, not an action) so a reflexive
    // Return can't confirm something destructive.
    <Dialog onClose={onCancel} role="alertdialog" initialFocus="dialog">
      <div className="w-full max-w-sm rounded-xl bg-raised border border-line shadow-2xl p-5">
        <h3 className="text-sm font-semibold text-ink">{options.title}</h3>
        {options.message && <p className="mt-1.5 text-xs text-ink-3">{options.message}</p>}
        <div className="mt-5 flex justify-end gap-2">
          <button
            type="button"
            onClick={onCancel}
            className="px-3 py-1.5 rounded-lg text-xs font-medium bg-fill text-ink-2 hover:bg-hover"
          >
            {t('actions.cancel')}
          </button>
          <button
            type="button"
            onClick={onConfirm}
            className={clsx(
              'px-3 py-1.5 rounded-lg text-xs font-medium text-white',
              options.destructive ? 'bg-red-600 hover:bg-red-700' : 'bg-accent-solid hover:bg-accent-solid-hover',
            )}
          >
            {options.confirmLabel}
          </button>
        </div>
      </div>
    </Dialog>
  )
}
