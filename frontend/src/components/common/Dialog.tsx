import clsx from 'clsx'
import { useEffect, useId, useLayoutEffect, useRef, useState, type KeyboardEvent, type ReactNode } from 'react'

const FOCUSABLE = [
  'a[href]',
  'button:not([disabled])',
  'input:not([disabled]):not([type="hidden"])',
  'select:not([disabled])',
  'textarea:not([disabled])',
  '[tabindex]:not([tabindex="-1"])',
].join(',')

const TITLE = 'h1, h2, h3, h4, [data-dialog-title]'

export interface DialogProps {
  onClose: () => void
  children: ReactNode
  // Classes for the dialog root -- usually the full-screen overlay
  // (positioning + backdrop); the panel markup lives in children.
  className?: string
  // Accessible name. Without one, the first heading inside names the dialog.
  label?: string
  // false while the dialog must not be dismissed (e.g. a save in flight).
  dismissible?: boolean
  // 'dialog' focuses the dialog itself rather than its first control, so
  // Return can't dismiss it by accident -- for alerts that must be read,
  // e.g. while a barcode scanner (types + Return) is in use.
  initialFocus?: 'first-control' | 'dialog'
}

function focusables(root: HTMLElement): HTMLElement[] {
  return Array.from(root.querySelectorAll<HTMLElement>(FOCUSABLE))
}

// Modal dialog behaviour for every overlay in the app: role="dialog" +
// aria-modal, a name, focus moved in on open and back to the opener on
// close, Tab kept inside, and Escape to dismiss (innermost dialog only).
export function Dialog({ onClose, children, className, label, dismissible = true, initialFocus = 'first-control' }: DialogProps) {
  const ref = useRef<HTMLDivElement>(null)
  const fallbackTitleId = useId()
  // Captured during render: an autoFocus child takes focus during commit,
  // before any effect of ours could see who had it.
  const [opener] = useState(() => (typeof document === 'undefined' ? null : document.activeElement as HTMLElement | null))

  useLayoutEffect(() => {
    const root = ref.current
    if (!root || label) return
    const title = root.querySelector<HTMLElement>(TITLE)
    if (!title) return
    if (!title.id) title.id = fallbackTitleId
    root.setAttribute('aria-labelledby', title.id)
  }, [label, fallbackTitleId])

  useEffect(() => {
    const root = ref.current
    if (root && !root.contains(document.activeElement)) {
      const first = initialFocus === 'first-control' ? focusables(root)[0] : undefined
      ;(first ?? root).focus()
    }
    return () => {
      if (opener && opener.isConnected) opener.focus()
    }
    // Initial focus is decided once, on open.
  }, [opener])

  function handleKeyDown(e: KeyboardEvent<HTMLDivElement>) {
    if (e.key === 'Escape') {
      e.stopPropagation()
      if (dismissible) onClose()
      return
    }
    if (e.key !== 'Tab' || !ref.current) return
    const items = focusables(ref.current)
    if (items.length === 0) {
      e.preventDefault()
      return
    }
    const first = items[0]
    const last = items[items.length - 1]
    if (e.shiftKey && document.activeElement === first) {
      e.preventDefault()
      last.focus()
    } else if (!e.shiftKey && document.activeElement === last) {
      e.preventDefault()
      first.focus()
    }
    e.stopPropagation()
  }

  return (
    <div
      ref={ref}
      role="dialog"
      aria-modal="true"
      aria-label={label}
      tabIndex={-1}
      onKeyDown={handleKeyDown}
      // The root only takes focus as a last resort (no focusable content).
      className={clsx(className ?? 'fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4', 'outline-none')}
    >
      {children}
    </div>
  )
}
