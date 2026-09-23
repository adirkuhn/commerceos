import { useCallback } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { idFromPath } from '../routes'
import { useStackNavigation } from './useStackNavigation'

export const NEW_RECORD = 'new'

export interface RoutedEditor {
  // null on the list, NEW_RECORD on `${basePath}/new`, else the record id.
  editingId: string | null
  openNew: () => void
  openEdit: (id: string) => void
  close: () => void
  // After creating from /new: swap the URL to the saved record's, so a
  // refresh shows the record and Back doesn't land on a blank create form.
  showSaved: (id: string) => void
}

// The "list page <-> full-page editor" pattern (coupons, promotions,
// warehouses, ...): which record is open lives in the URL --
// `${basePath}/new` or `${basePath}/:id` -- instead of useState, so the
// editor survives a refresh and the browser Back button closes it.
export function useRoutedEditor(basePath: string): RoutedEditor {
  const location = useLocation()
  const navigate = useNavigate()
  const { open, exitToOpener } = useStackNavigation()
  const editingId = idFromPath(location.pathname, basePath)
  const pathFor = useCallback((id: string) => `${basePath}/${encodeURIComponent(id)}`, [basePath])

  return {
    editingId,
    openNew: useCallback(() => open(pathFor(NEW_RECORD)), [open, pathFor]),
    openEdit: useCallback((id: string) => open(pathFor(id)), [open, pathFor]),
    close: useCallback(() => exitToOpener(basePath), [exitToOpener, basePath]),
    showSaved: useCallback(
      (id: string) => navigate(pathFor(id), { replace: true, state: location.state }),
      [navigate, pathFor, location.state],
    ),
  }
}
