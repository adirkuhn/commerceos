import { useState } from 'react'
import { Bot, X } from 'lucide-react'
import clsx from 'clsx'
import { useTranslation } from 'react-i18next'
import type { TFunction } from 'i18next'
import { connectionStatusLabel } from '../integrations/connectionStatus'
import { useAuth } from '../auth/AuthContext'
import { useAIConnection } from '../integrations/hooks/useAIConnection'
import type { AIProvider } from '../integrations/types'
import { Dialog } from '../components/common/Dialog'
import { useConfirm } from '../components/common/useConfirm'
import { STATUS_STYLES } from './IntegrationsSettingsSection'

const AI_PROVIDERS: AIProvider[] = ['openai', 'anthropic', 'gemini', 'ollama']

function providerLabel(t: TFunction<['settings', 'common']>, provider: AIProvider): string {
  switch (provider) {
    case 'openai': return 'OpenAI'
    case 'anthropic': return 'Anthropic (Claude)'
    case 'gemini': return 'Google Gemini'
    case 'ollama': return t('ai.ollama')
  }
}

// AI is a separate integration shape from Connection (WooCommerce etc.) --
// one row per tenant, no store URL/OAuth, just a provider + credential --
// so it gets its own card and configure flow rather than joining the
// Connection list above.
export function AIIntegrationCard() {
  const { confirm, confirmDialog } = useConfirm()
  const { t } = useTranslation(['settings', 'common'])
  const { token } = useAuth()
  const { connection, loading, error, connect, disconnect } = useAIConnection(token ?? '')
  const [configuring, setConfiguring] = useState(false)
  const [provider, setProvider] = useState<AIProvider>('openai')
  const [apiKey, setApiKey] = useState('')
  const [baseUrl, setBaseUrl] = useState('')
  const [model, setModel] = useState('')
  const [fallbackModels, setFallbackModels] = useState('')
  const [saving, setSaving] = useState(false)
  const [formError, setFormError] = useState('')

  const openConfigure = () => {
    setProvider(connection?.provider ?? 'openai')
    setApiKey('')
    setBaseUrl(connection?.base_url ?? '')
    setModel(connection?.model ?? '')
    setFallbackModels(connection?.fallback_models ?? '')
    setFormError('')
    setConfiguring(true)
  }

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault()
    setSaving(true)
    setFormError('')
    try {
      await connect({ provider, apiKey, baseUrl, model, fallbackModels })
      setConfiguring(false)
    } catch (err) {
      setFormError(err instanceof Error ? err.message : t('ai.connectFailed'))
    } finally {
      setSaving(false)
    }
  }

  const handleDisconnect = async () => {
    if (!(await confirm({ title: t('ai.disconnectTitle'), message: t('ai.disconnectMessage'), confirmLabel: t('ai.disconnect'), destructive: true }))) return
    await disconnect()
  }

  return (
    <div className="bg-raised rounded-xl border border-line p-4">
      <div className="flex items-center justify-between gap-4">
        <div className="min-w-0 flex items-center gap-3">
          <div className="w-8 h-8 rounded-lg bg-gradient-to-br from-blue-500 to-cyan-600 flex items-center justify-center text-white shrink-0">
            <Bot size={14} />
          </div>
          <div className="min-w-0">
            <div className="flex items-center gap-2">
              <p className="text-xs font-medium text-ink">{t('ai.title')}</p>
              {connection && (
                <span className={clsx('px-1.5 py-0.5 rounded-full text-2xs font-medium capitalize', STATUS_STYLES[connection.status])}>
                  {connectionStatusLabel(connection.status)}
                </span>
              )}
            </div>
            <p className="text-2xs text-ink-4 mt-0.5">
              {loading ? t('common:status.loading') : connection
                ? providerLabel(t, connection.provider)
                : t('ai.notConfigured')}
            </p>
          </div>
        </div>
        <div className="flex items-center gap-2 shrink-0">
          <button
            onClick={openConfigure}
            className="px-3 py-1.5 rounded-lg bg-fill text-ink-2 text-xs font-medium hover:bg-hover transition-colors"
          >
            {connection ? t('integrations.configure') : t('ai.connect')}
          </button>
          {connection && (
            <button
              onClick={handleDisconnect}
              className="px-3 py-1.5 rounded-lg bg-fill text-xs font-medium text-ink-3 hover:text-danger transition-colors"
            >
              {t('ai.disconnect')}
            </button>
          )}
        </div>
      </div>
      {error && <p className="mt-2 text-2xs text-danger">{error}</p>}

      {configuring && (
        <Dialog onClose={() => setConfiguring(false)} className="fixed inset-0 z-50 flex items-center justify-center">
          <div className="absolute inset-0 dark:bg-black/60 bg-black/30" onClick={() => setConfiguring(false)} />
          <div className="relative w-full max-w-sm bg-raised rounded-xl shadow-2xl border border-line p-5">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-sm font-semibold text-ink">{t('ai.configureTitle')}</h3>
              <button onClick={() => setConfiguring(false)} aria-label={t('common:actions.close')}>
                <X size={14} className="text-ink-4" />
              </button>
            </div>
            <form onSubmit={handleSave} className="space-y-3">
              <div>
                <label htmlFor="ai-provider" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
                  {t('ai.provider')}
                </label>
                <select
                  id="ai-provider"
                  value={provider}
                  onChange={e => setProvider(e.target.value as AIProvider)}
                  className="w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus"
                >
                  {AI_PROVIDERS.map(p => (
                    <option key={p} value={p}>{providerLabel(t, p)}</option>
                  ))}
                </select>
              </div>
              {provider !== 'ollama' && (
                <div>
                  <label htmlFor="ai-api-key" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
                    {t('ai.apiKey')}
                  </label>
                  <input
                    id="ai-api-key"
                    type="password"
                    required
                    autoFocus
                    value={apiKey}
                    onChange={e => setApiKey(e.target.value)}
                    placeholder={'sk-...'}
                    className="w-full px-3 py-2 text-xs font-mono rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus"
                  />
                </div>
              )}
              <div>
                <label htmlFor="ai-base-url" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
                  {provider === 'ollama' ? t('ai.serverUrl') : t('ai.baseUrl')}
                </label>
                <input
                  id="ai-base-url"
                  type="url"
                  required={provider === 'ollama'}
                  autoFocus={provider === 'ollama'}
                  value={baseUrl}
                  onChange={e => setBaseUrl(e.target.value)}
                  placeholder={provider === 'ollama' ? 'http://localhost:11434' : 'https://...'}
                  className="w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus"
                />
              </div>
              <div>
                <label htmlFor="ai-model" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
                  {t('ai.model')}
                </label>
                <input
                  id="ai-model"
                  type="text"
                  value={model}
                  onChange={e => setModel(e.target.value)}
                  placeholder={t('ai.modelPlaceholder')}
                  className="w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus"
                />
              </div>
              {provider === 'gemini' && (
                <div>
                  <label htmlFor="ai-fallback-models" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
                    {t('ai.fallbackModels')}
                  </label>
                  <input
                    id="ai-fallback-models"
                    type="text"
                    value={fallbackModels}
                    onChange={e => setFallbackModels(e.target.value)}
                    placeholder={'gemini-3.5-flash-lite, gemini-3.1-flash-lite'}
                    className="w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus"
                  />
                  <p className="mt-1.5 text-2xs text-ink-4">
                    {t('ai.fallbackHelp')}
                  </p>
                </div>
              )}
              {formError && (
                <p className="text-xs text-danger bg-red-50 dark:bg-red-500/10 rounded-lg px-2 py-1.5">{formError}</p>
              )}
              <button
                type="submit"
                disabled={saving}
                className="w-full py-2 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover disabled:opacity-50 transition-colors"
              >
                {saving ? t('common:actions.saving') : t('common:actions.save')}
              </button>
            </form>
          </div>
        </Dialog>
      )}
      {confirmDialog}
    </div>
  )
}
