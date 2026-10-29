import { useEffect, type ReactNode } from 'react'
import { createPortal } from 'react-dom'

// The printable page every print view (invoice, picking lists, supplier
// issues) renders into. On screen it's a full-window overlay. For print it
// must escape the app shell: the shell is screen-height with overflow
// hidden, and a fixed/scrolling overlay prints only what fits on one
// screen -- page 1 and nothing after. So the sheet is portalled straight
// under <body>, and while it's open body.printing lets index.css hide
// everything else and let the sheet flow over as many pages as it needs,
// with the @page margins applied to every page (see index.css).
export function PrintSheet({ children }: { children: ReactNode }) {
  useEffect(() => {
    document.body.classList.add('printing')
    return () => { document.body.classList.remove('printing') }
  }, [])

  return createPortal(
    <div className="print-sheet fixed inset-0 z-50 bg-white overflow-y-auto">{children}</div>,
    document.body,
  )
}
