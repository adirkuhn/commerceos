import { useCallback } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { useStackNavigation } from '../routing/useStackNavigation'
import { INVENTORY_PATH, inventoryPaths, parseInventoryPath, type SkuAction } from './inventoryRoutes'

// Which batch row an action is locked to: a real batch by id, or a
// warehouse's general-ledger row (which has no batch id of its own).
export type BatchRowRef = { batch: string } | { ledger: string }

const SCOPE_PARAMS = ['batch', 'ledger'] as const

function withoutScope(search: string): URLSearchParams {
  const params = new URLSearchParams(search)
  for (const key of SCOPE_PARAMS) params.delete(key)
  return params
}

function withRowRef(params: URLSearchParams, row?: BatchRowRef): URLSearchParams {
  const next = new URLSearchParams(params)
  if (row && 'batch' in row) next.set('batch', row.batch)
  if (row && 'ledger' in row) next.set('ledger', row.ledger)
  return next
}

// Same encoding as useQuerySet's comma-separated sets.
function addToSet(params: URLSearchParams, key: string, id: string) {
  const values = new Set((params.get(key) ?? '').split(',').filter(Boolean))
  values.add(encodeURIComponent(id))
  params.set(key, [...values].join(','))
}

function withSearch(path: string, params: URLSearchParams): string {
  const qs = params.toString()
  return qs ? `${path}?${qs}` : path
}

export interface SkuActionRoute {
  // The action open for this SKU (null when none, or when the URL names a
  // different SKU).
  action: SkuAction | null
  row: BatchRowRef | null
  openAction: (action: SkuAction, row?: BatchRowRef) => void
  // From the batch-row chooser to the picked action: replaces, so Back
  // closes the action rather than reopening the chooser.
  switchAction: (action: SkuAction) => void
  close: () => void
  // After creating a batch: back to the list with this SKU's zero-stock
  // batches shown (?allBatches=), since a new batch starts at zero and
  // would otherwise be hidden the moment the modal closes.
  closeShowingAllBatches: () => void
}

// SKU action modals (Adjust, Recount, ... -- see inventoryRoutes) live at
// /inventory/products/<p>/skus/<s>/<action>, carrying the list's own query
// (?open=, ?q=, ?page=, ...) so the list underneath looks the same after a
// refresh, plus ?batch= / ?ledger= when opened from a batch row.
export function useSkuActionRoute(productId: string, skuId: string): SkuActionRoute {
  const location = useLocation()
  const navigate = useNavigate()
  const { open, exitToOpener } = useStackNavigation()
  const view = parseInventoryPath(location.pathname)
  const mine = view.kind === 'skuAction' && view.productId === productId && view.skuId === skuId
  const params = new URLSearchParams(location.search)
  const batch = params.get('batch')
  const ledger = params.get('ledger')

  const openAction = useCallback((action: SkuAction, row?: BatchRowRef) => {
    open(withSearch(inventoryPaths.skuAction(productId, skuId, action), withRowRef(withoutScope(location.search), row)))
  }, [open, productId, skuId, location.search])

  const switchAction = useCallback((action: SkuAction) => {
    navigate(`${inventoryPaths.skuAction(productId, skuId, action)}${location.search}`, { replace: true, state: location.state })
  }, [navigate, productId, skuId, location.search, location.state])

  const listUrl = useCallback((showAllBatches: boolean) => {
    const list = withoutScope(location.search)
    addToSet(list, 'open', productId)
    if (showAllBatches) addToSet(list, 'allBatches', skuId)
    return withSearch(INVENTORY_PATH, list)
  }, [location.search, productId, skuId])

  // Deep-linked (no opener): fall back to the list with this product's row
  // kept open, so closing the modal doesn't also collapse what it was about.
  const close = useCallback(() => exitToOpener(listUrl(false)), [exitToOpener, listUrl])

  const closeShowingAllBatches = useCallback(() => {
    navigate(listUrl(true), { replace: true })
  }, [navigate, listUrl])

  return {
    action: mine ? view.action : null,
    row: batch !== null ? { batch } : ledger !== null ? { ledger } : null,
    openAction,
    switchAction,
    close,
    closeShowingAllBatches,
  }
}
