import { useTranslation } from 'react-i18next'
import { useAuth } from '../auth/AuthContext'
import { useApp } from '../context/AppContext'
import { useConnections } from '../integrations/hooks/useConnections'
import { useWebhookFallbackEvents } from '../integrations/hooks/useWebhookFallbackEvents'
import { useTestConnection } from '../integrations/hooks/useTestConnection'
import { WebhookFallbackEventsTable } from '../integrations/WebhookFallbackEventsTable'
import { WhatsAppTestModeModal } from '../integrations/WhatsAppTestModeModal'
import { WhatsAppAIParsingModal } from '../integrations/WhatsAppAIParsingModal'
import { WhatsAppCredentialsModal } from '../integrations/WhatsAppCredentialsModal'
import { WhatsAppWebhookModal } from '../integrations/WhatsAppWebhookModal'
import { BrowseIntegrationsDialog } from '../integrations/BrowseIntegrationsDialog'
import { ImportProductsDialog } from '../integrations/ImportProductsDialog'
import { DeleteImportedProductsDialog } from '../integrations/DeleteImportedProductsDialog'
import { ConnectionsOverview, WebhooksShortcut } from '../integrations/ConnectionsOverview'
import type { Connection } from '../integrations/types'
import { useWarehouses } from '../inventory/hooks/useWarehouses'
import { useTenantSettings } from '../settings/hooks/useTenantSettings'
import WooCommerceSyncSettingsPage from '../integrations/WooCommerceSyncSettingsPage'
import BolSyncSettingsPage from '../integrations/BolSyncSettingsPage'
import MyParcelSyncSettingsPage from '../shipping/MyParcelSyncSettingsPage'
import { useIntegrationsRoute, type ConnectionPanel } from '../integrations/hooks/useIntegrationsRoute'
import { RecordLoading, RecordNotFound } from '../components/common/RecordStatus'
import { canManageWebhooks } from '../integrations/integrationsPageHelpers'
import { DefaultShippingProviderSetting } from '../shipping/DefaultShippingProviderSetting'
import { ShippingAddressesSetting } from '../shipping/addresses/ShippingAddressesSetting'
import { isTenantAdmin } from '../auth/roles'

export default function Integrations() {
  const { t } = useTranslation('integrations')
  const { token, user } = useAuth()
  const authToken = token ?? ''
  const { connections, loading, error, refresh, disconnect } = useConnections(authToken)
  const { warehouses } = useWarehouses(authToken)
  const { settings: tenantSettings } = useTenantSettings(authToken)
  const showWebhookAdmin = canManageWebhooks(user?.role)
  const { setActiveNav } = useApp()
  const fallbackEvents = useWebhookFallbackEvents(showWebhookAdmin ? authToken : '')
  const { testingId, testResult, test } = useTestConnection(authToken, refresh)

  // Which page/modal is open -- a connection's settings page or one of its
  // modals, the "Add integration" catalog, a connect form -- comes from the
  // URL (see useIntegrationsRoute), so a refresh keeps it open and Back
  // closes it. Form field values stay local (in each modal).
  const route = useIntegrationsRoute()
  const { view } = route
  const routeConnectionId = view.kind === 'connection' ? view.connectionId : null
  const routeConnection = routeConnectionId ? connections.find(c => c.id === routeConnectionId) ?? null : null
  const panelFor = (panel: ConnectionPanel, providers?: string[]): Connection | null =>
    view.kind === 'connection' && view.panel === panel && routeConnection && (!providers || providers.includes(routeConnection.provider))
      ? routeConnection
      : null

  const configuringConnection = panelFor('settings', ['woocommerce'])
  const configuringWhatsAppTestMode = panelFor('test-mode')
  const configuringWhatsAppAIParsing = panelFor('ai-parsing')
  const configuringWhatsAppCredentials = panelFor('credentials')
  const configuringWhatsAppWebhook = panelFor('whatsapp-webhook')
  const importingConnection = panelFor('import')
  const deletingConnection = panelFor('delete-products')

  const browsing = view.kind === 'browse' || view.kind === 'connect'
  const bolConnection: Connection | 'new' | null =
    view.kind === 'new' && view.provider === 'bol' ? 'new' : panelFor('settings', ['bol'])
  const myParcelConnection: Connection | 'new' | null =
    view.kind === 'new' && view.provider === 'myparcel' ? 'new' : panelFor('settings', ['myparcel']) ?? panelFor('webhooks', ['myparcel'])
  const focusMyParcelWebhooks = view.kind === 'connection' && view.panel === 'webhooks'

  const closeAndRefresh = () => {
    route.close()
    refresh()
  }

  if (routeConnectionId && !routeConnection) {
    return loading ? <RecordLoading /> : <RecordNotFound label={t('connection')} onBack={route.close} />
  }

  if (configuringConnection) {
    return <WooCommerceSyncSettingsPage connection={configuringConnection} onClose={route.close} />
  }

  if (bolConnection) {
    return (
      <BolSyncSettingsPage
        connectionId={bolConnection === 'new' ? undefined : bolConnection.id}
        initialConnection={bolConnection === 'new' ? undefined : bolConnection}
        onClose={route.close}
        onConnected={closeAndRefresh}
      />
    )
  }

  if (myParcelConnection) {
    return (
      <MyParcelSyncSettingsPage
        connectionId={myParcelConnection === 'new' ? undefined : myParcelConnection.id}
        focusWebhooks={focusMyParcelWebhooks}
        onClose={route.close}
        onConnected={closeAndRefresh}
      />
    )
  }

  return (
    <div className="flex h-full overflow-hidden">
      <div className="flex-1 flex flex-col overflow-hidden p-5 gap-5">
        {showWebhookAdmin && <WebhooksShortcut onOpen={() => setActiveNav('webhooks')} />}

        {/* Mounted once the connection list is in, keyed on it, so
            connecting/disconnecting one reloads which shipping providers
            it offers. */}
        {!loading && (
          <>
            <DefaultShippingProviderSetting
              key={connections.map(c => c.id).join(',')}
              token={authToken}
              canEdit={isTenantAdmin(user?.role)}
            />
            <ShippingAddressesSetting
              key={`addresses-${connections.map(c => c.id).join(',')}`}
              token={authToken}
              canEdit={isTenantAdmin(user?.role)}
            />
          </>
        )}

        <ConnectionsOverview
          connections={connections}
          loading={loading}
          error={error}
          testingId={testingId}
          testResult={testResult}
          onBrowse={route.openBrowse}
          onOpenPanel={(conn, panel) => route.openPanel(conn.id, panel)}
          onTest={test}
          onDisconnect={disconnect}
        />

        {showWebhookAdmin && <WebhookFallbackEventsTable {...fallbackEvents} />}
      </div>

      {configuringWhatsAppTestMode && (
        <WhatsAppTestModeModal
          connection={configuringWhatsAppTestMode}
          onClose={route.close}
          onSaved={() => refresh()}
        />
      )}

      {configuringWhatsAppAIParsing && (
        <WhatsAppAIParsingModal
          connection={configuringWhatsAppAIParsing}
          onClose={route.close}
          onSaved={() => refresh()}
        />
      )}

      {configuringWhatsAppCredentials && (
        <WhatsAppCredentialsModal
          connection={configuringWhatsAppCredentials}
          onClose={route.close}
          onSaved={() => refresh()}
        />
      )}

      {configuringWhatsAppWebhook && (
        <WhatsAppWebhookModal
          connection={configuringWhatsAppWebhook}
          tenantId={user?.tenant_id ?? ''}
          onClose={route.close}
        />
      )}

      {browsing && (
        <BrowseIntegrationsDialog
          token={authToken}
          provider={view.kind === 'connect' ? view.provider : null}
          onOpenConnect={route.openConnect}
          onOpenNew={route.openNew}
          onConnected={refresh}
          onClose={route.close}
        />
      )}

      {importingConnection && (
        <ImportProductsDialog
          token={authToken}
          connection={importingConnection}
          warehouses={warehouses}
          tenantDefaultCurrency={tenantSettings?.default_currency}
          onClose={route.close}
        />
      )}

      {deletingConnection && (
        <DeleteImportedProductsDialog token={authToken} connection={deletingConnection} onClose={closeAndRefresh} />
      )}
    </div>
  )
}
