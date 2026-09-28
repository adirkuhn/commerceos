import { useCallback } from 'react'
import { useLocation } from 'react-router-dom'
import { subPathSegments } from '../../routes'
import { useStackNavigation } from '../../routing/useStackNavigation'
import { NEW_RECORD } from '../../routing/useRoutedEditor'

export const SALES_CHANNELS_PATH = '/sales-channels'

export type SalesChannelView =
  | { kind: 'list' }
  | { kind: 'create' }
  | { kind: 'manage'; channelId: string }
  | { kind: 'edit'; channelId: string }
  | { kind: 'api'; channelId: string }
  | { kind: 'themes'; channelId: string }

// Which Sales Channels view is showing, derived from the URL:
//   /sales-channels              list
//   /sales-channels/new          create form (modal over the list)
//   /sales-channels/:id          products & pricing manager
//   /sales-channels/:id/edit     edit form (modal over the list)
//   /sales-channels/:id/api      headless API docs (modal over the list)
//   /sales-channels/:id/themes   theme library (internal channels)
// so each survives a refresh and Back closes it.
export function useSalesChannelRoute() {
  const location = useLocation()
  const { open, exitToOpener } = useStackNavigation()
  const [first, second] = subPathSegments(location.pathname, SALES_CHANNELS_PATH)

  let view: SalesChannelView
  if (!first) view = { kind: 'list' }
  else if (first === NEW_RECORD) view = { kind: 'create' }
  else if (second === 'edit') view = { kind: 'edit', channelId: first }
  else if (second === 'api') view = { kind: 'api', channelId: first }
  else if (second === 'themes') view = { kind: 'themes', channelId: first }
  else view = { kind: 'manage', channelId: first }

  const channelPath = (id: string) => `${SALES_CHANNELS_PATH}/${encodeURIComponent(id)}`

  return {
    view,
    openCreate: useCallback(() => open(`${SALES_CHANNELS_PATH}/${NEW_RECORD}`), [open]),
    openManage: useCallback((id: string) => open(channelPath(id)), [open]),
    openEdit: useCallback((id: string) => open(`${channelPath(id)}/edit`), [open]),
    openApiDocs: useCallback((id: string) => open(`${channelPath(id)}/api`), [open]),
    openThemes: useCallback((id: string) => open(`${channelPath(id)}/themes`), [open]),
    close: useCallback(() => exitToOpener(SALES_CHANNELS_PATH), [exitToOpener]),
  }
}
