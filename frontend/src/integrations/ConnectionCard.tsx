import clsx from 'clsx'
import i18next from 'i18next'
import { useTranslation } from 'react-i18next'
import type { Connection } from './types'
import type { ConnectionPanel } from './hooks/useIntegrationsRoute'
import type { TestResult } from './hooks/useTestConnection'
import { PROVIDER_COLORS, STATUS_COLOR, STATUS_ICON } from './integrationsPageHelpers'

type CardAction =
  | { kind: 'panel'; label: () => string; panel: ConnectionPanel; danger?: boolean }
  | { kind: 'test' }

const panel = (label: () => string, target: ConnectionPanel, danger = false): CardAction => ({ kind: 'panel', label, panel: target, danger })

const configure = () => i18next.t('integrations:card.configure')

// The buttons each provider's card offers, in display order ("Disconnect"
// is always appended). Providers not listed only get Disconnect.
export const PROVIDER_CARD_ACTIONS: Record<string, CardAction[]> = {
  woocommerce: [panel(configure, 'settings'), { kind: 'test' }, panel(() => i18next.t('common:actions.import'), 'import'), panel(() => i18next.t('integrations:card.deleteImported'), 'delete-products', true)],
  myparcel: [panel(configure, 'settings'), panel(() => i18next.t('integrations:card.registerWebhook'), 'webhooks')],
  bol: [panel(configure, 'settings')],
  whatsapp: [
    panel(() => i18next.t('integrations:card.testMode'), 'test-mode'),
    panel(() => i18next.t('integrations:card.aiParsing'), 'ai-parsing'),
    panel(() => i18next.t('integrations:card.editCredentials'), 'credentials'),
    panel(() => i18next.t('integrations:card.webhook'), 'whatsapp-webhook'),
  ],
}

interface Props {
  connection: Connection
  testing: boolean
  // The last Test result for this connection, if any.
  testResult: TestResult | null
  onOpenPanel: (panel: ConnectionPanel) => void
  onTest: () => void
  onDisconnect: () => void
}

export function ConnectionCard({ connection: conn, testing, testResult, onOpenPanel, onTest, onDisconnect }: Props) {
  const { t } = useTranslation('integrations')
  const Icon = STATUS_ICON[conn.status]
  const actions = PROVIDER_CARD_ACTIONS[conn.provider] ?? []
  return (
    <div
      className="text-left p-4 rounded-xl border border-line bg-raised"
    >
      <div className="flex items-start justify-between mb-3">
        <div className={clsx(
          'w-9 h-9 rounded-lg bg-gradient-to-br flex items-center justify-center text-white font-bold text-sm',
          PROVIDER_COLORS[conn.provider],
        )}>
          {conn.provider[0].toUpperCase()}
        </div>
        <Icon size={14} className={STATUS_COLOR[conn.status]} />
      </div>
      <p className="text-sm font-semibold text-ink capitalize">{conn.provider}</p>
      <p className="text-2xs text-ink-4 mt-0.5 truncate">{conn.store_url}</p>
      {conn.provider === 'whatsapp' && conn.whatsapp_test_mode_enabled && (
        <p className="mt-2 text-2xs px-2 py-1 rounded-lg text-warning bg-amber-500/10">
          {t('card.testModeOn', { phone: conn.whatsapp_test_phone_e164 })}
        </p>
      )}
      {conn.provider === 'whatsapp' && conn.whatsapp_ai_parsing_enabled && (
        <p className="mt-2 text-2xs px-2 py-1 rounded-lg text-accent bg-accent/10">
          {t('card.aiParsingOn')}
        </p>
      )}
      {testResult && (
        <p className={clsx(
          'mt-2 text-2xs px-2 py-1 rounded-lg',
          testResult.ok ? 'text-success bg-green-500/10' : 'text-danger bg-red-500/10',
        )}>
          {testResult.message}
        </p>
      )}
      <div className="mt-3 grid grid-cols-2 gap-2">
        {actions.map(action => action.kind === 'test' ? (
          <button
            key="test"
            onClick={onTest}
            disabled={testing}
            className="py-1.5 rounded-lg bg-fill text-2xs text-ink-3 hover:text-accent disabled:opacity-50 transition-colors"
          >
            {testing ? t('card.testing') : t('card.test')}
          </button>
        ) : (
          <button
            key={action.panel}
            onClick={() => onOpenPanel(action.panel)}
            className={clsx('py-1.5 rounded-lg bg-fill text-2xs text-ink-3', action.danger ? 'hover:text-danger' : 'hover:text-accent', 'transition-colors')}
          >
            {action.label()}
          </button>
        ))}
        <button
          onClick={onDisconnect}
          className="py-1.5 rounded-lg bg-fill text-2xs text-ink-3 hover:text-danger transition-colors"
        >
          {t('card.disconnect')}
        </button>
      </div>
    </div>
  )
}
