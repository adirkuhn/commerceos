import clsx from 'clsx'
import { useTranslation } from 'react-i18next'
import { connectionStatusLabel } from '../integrations/connectionStatus'
import { useAuth } from '../auth/AuthContext'
import { useConnections } from '../integrations/hooks/useConnections'
import type { Connection } from '../integrations/types'
import { AIIntegrationCard } from './AIIntegrationCard'
import { ApiKeysCard } from './ApiKeysCard'
import { RecommendationsSettingsCard } from './RecommendationsSettingsCard'

// ── Integrations settings section ─────────────────────────────────────────────

export const STATUS_STYLES: Record<Connection['status'], string> = {
  connected: 'bg-green-500/15 text-success',
  error: 'bg-red-500/15 text-danger',
}

export function IntegrationsSettingsSection({ onConfigure }: { onConfigure: (connection: Connection) => void }) {
  const { token } = useAuth()
  const { t } = useTranslation(['settings', 'common'])
  const { connections, loading, error } = useConnections(token ?? '')

  return (
    <div className="max-w-2xl space-y-6">
      <div>
        <h2 className="text-base font-semibold text-ink mb-1">{t('sections.integrations')}</h2>
        <p className="text-xs text-ink-4">{t('integrations.subtitle')}</p>
      </div>

      <AIIntegrationCard />

      <ApiKeysCard />

      <RecommendationsSettingsCard />

      {loading && <p className="text-xs text-ink-4">{t('common:status.loading')}</p>}
      {error && <p className="text-xs text-danger">{error}</p>}

      {!loading && connections.length === 0 && (
        <div className="bg-raised rounded-xl border border-line p-6 text-center">
          <p className="text-xs text-ink-4">{t('integrations.none')}</p>
        </div>
      )}

      <div className="space-y-3">
        {connections.map(conn => (
          <div key={conn.id} className="bg-raised rounded-xl border border-line p-4 flex items-center justify-between gap-4">
            <div className="min-w-0">
              <div className="flex items-center gap-2">
                <p className="text-xs font-medium text-ink capitalize">{conn.provider}</p>
                <span className={clsx('px-1.5 py-0.5 rounded-full text-2xs font-medium capitalize', STATUS_STYLES[conn.status])}>
                  {connectionStatusLabel(conn.status)}
                </span>
              </div>
              <p className="text-2xs text-ink-4 truncate mt-0.5">{conn.store_url}</p>
            </div>
            {conn.provider === 'woocommerce' ? (
              <button
                onClick={() => onConfigure(conn)}
                className="px-3 py-1.5 rounded-lg bg-fill text-ink-2 text-xs font-medium hover:bg-hover transition-colors shrink-0"
              >
                {t('integrations.configure')}
              </button>
            ) : (
              <span className="text-2xs text-ink-4 shrink-0">{t('integrations.comingSoon')}</span>
            )}
          </div>
        ))}
      </div>
    </div>
  )
}
