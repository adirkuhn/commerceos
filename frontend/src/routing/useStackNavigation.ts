import { useCallback } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'

// History-stack aware navigation between a parent screen (a list, a
// launcher) and a child screen (a record, a session).
//
// Why not plain navigate(parent) on exit: that pushes the parent on top of
// the child, so the device/browser Back button then re-enters the child the
// user just left. Why not plain navigate(-1): a deep-linked or refreshed
// child has no parent entry behind it, and -1 would leave the app.
//
// open() records the URL it was opened from in history state (which the
// browser keeps across a reload); exitTo() pops when that recorded opener
// is the requested parent and replaces otherwise.

interface OpenerState {
  openedFrom?: string
}

export interface StackNavigation {
  open: (to: string) => void
  exitTo: (parent: string) => void
  // Like exitTo, but returns to the exact opener URL (e.g. the list with its
  // filters/page) whenever there is one, falling back to `parent`.
  exitToOpener: (parent: string) => void
}

function openerOf(state: unknown): string | undefined {
  return (state as OpenerState | null)?.openedFrom
}

export function useStackNavigation(): StackNavigation {
  const navigate = useNavigate()
  const location = useLocation()
  const here = location.pathname + location.search
  const opener = openerOf(location.state)

  const open = useCallback((to: string) => {
    navigate(to, { state: { openedFrom: here } satisfies OpenerState })
  }, [navigate, here])

  const exitTo = useCallback((parent: string) => {
    if (opener === parent) navigate(-1)
    else navigate(parent, { replace: true })
  }, [navigate, opener])

  const exitToOpener = useCallback((parent: string) => {
    if (opener !== undefined) navigate(-1)
    else navigate(parent, { replace: true })
  }, [navigate, opener])

  return { open, exitTo, exitToOpener }
}
