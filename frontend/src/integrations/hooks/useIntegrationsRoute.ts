import { useCallback } from 'react'
import { useLocation } from 'react-router-dom'
import { subPathSegments } from '../../routes'
import { useStackNavigation } from '../../routing/useStackNavigation'
import { NEW_RECORD } from '../../routing/useRoutedEditor'
import { INTEGRATIONS_PATH } from '../integrationsPaths'

export { INTEGRATIONS_PATH }
const CONNECT_SEGMENT = 'connect'

// Providers connected through a form in the "Add integration" catalog.
export const CONNECT_FORM_PROVIDERS = ['woocommerce', 'whatsapp', 'stripe', 'revolut', 'email', 'postnl'] as const
export type ConnectFormProvider = typeof CONNECT_FORM_PROVIDERS[number]

// Providers whose "connect" is their full settings page, opened blank.
export const NEW_PAGE_PROVIDERS = ['bol', 'myparcel'] as const
export type NewPageProvider = typeof NEW_PAGE_PROVIDERS[number]

// What can be open for one existing connection. 'settings' is the
// provider's own settings page (WooCommerce / bol / MyParcel); the rest
// are modals over the list.
export const CONNECTION_PANELS = [
  'settings', 'webhooks', 'test-mode', 'ai-parsing', 'credentials', 'whatsapp-webhook', 'import', 'delete-products',
] as const
export type ConnectionPanel = typeof CONNECTION_PANELS[number]
const DEFAULT_PANEL: ConnectionPanel = 'settings'

export type IntegrationsView =
  | { kind: 'list' }
  | { kind: 'browse' }
  | { kind: 'connect'; provider: ConnectFormProvider }
  | { kind: 'new'; provider: NewPageProvider }
  | { kind: 'connection'; connectionId: string; panel: ConnectionPanel }

function isOneOf<T extends string>(values: readonly T[], v: string | undefined): v is T {
  return v !== undefined && (values as readonly string[]).includes(v)
}

// Which Integrations page/modal is open, from the URL:
//   /integrations                          list
//   /integrations/connect[/<provider>]     "Add integration" catalog / connect form
//   /integrations/new/<bol|myparcel>       blank provider settings page
//   /integrations/<connectionId>[/<panel>] a connection's settings page or modal
// so each survives a refresh and Back closes it.
export function useIntegrationsRoute() {
  const location = useLocation()
  const { open, exitTo, exitToOpener } = useStackNavigation()
  const [first, second] = subPathSegments(location.pathname, INTEGRATIONS_PATH)

  let view: IntegrationsView
  if (!first) view = { kind: 'list' }
  else if (first === CONNECT_SEGMENT) {
    view = isOneOf(CONNECT_FORM_PROVIDERS, second) ? { kind: 'connect', provider: second } : { kind: 'browse' }
  } else if (first === NEW_RECORD) {
    view = isOneOf(NEW_PAGE_PROVIDERS, second) ? { kind: 'new', provider: second } : { kind: 'list' }
  } else {
    view = { kind: 'connection', connectionId: first, panel: isOneOf(CONNECTION_PANELS, second) ? second : DEFAULT_PANEL }
  }

  const connectPath = `${INTEGRATIONS_PATH}/${CONNECT_SEGMENT}`

  return {
    view,
    openBrowse: useCallback(() => open(connectPath), [open, connectPath]),
    openConnect: useCallback((provider: ConnectFormProvider) => open(`${connectPath}/${provider}`), [open, connectPath]),
    openNew: useCallback((provider: NewPageProvider) => open(`${INTEGRATIONS_PATH}/${NEW_RECORD}/${provider}`), [open]),
    openPanel: useCallback((connectionId: string, panel: ConnectionPanel) => {
      const base = `${INTEGRATIONS_PATH}/${encodeURIComponent(connectionId)}`
      open(panel === DEFAULT_PANEL ? base : `${base}/${panel}`)
    }, [open]),
    backToBrowse: useCallback(() => exitTo(connectPath), [exitTo, connectPath]),
    close: useCallback(() => exitToOpener(INTEGRATIONS_PATH), [exitToOpener]),
  }
}
