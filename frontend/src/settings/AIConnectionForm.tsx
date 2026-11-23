import { useState } from 'react'
import { X } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import type { AIConnection, AIProvider } from '../integrations/types'
import { Dialog } from '../components/common/Dialog'
import { apiErrorText } from '../api/apiErrorText'
import {
  AI_PROVIDERS, API_KEY_PLACEHOLDER, DEFAULT_MODEL, FALLBACK_PLACEHOLDER, PROVIDERS_WITH_FALLBACK, SELF_HOSTED, providerLabel,
} from './aiProviders'

export interface AIConnectionFormValues {
  provider: AIProvider
  apiKey: string
  baseUrl: string
  model: string
  fallbackModels: string
  label: string
}

const INPUT = 'w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus'
const LABEL = 'block text-2xs text-ink-4 caps-label mb-1.5'

// Adds a connection (initial null) or edits one. Editing never shows the
// stored key: leaving the field empty keeps it.
export function AIConnectionForm({ initial, onSave, onClose }: {
  initial: AIConnection | null
  onSave: (values: AIConnectionFormValues) => Promise<void>
  onClose: () => void
}) {
  const { t } = useTranslation(['settings', 'common'])
  const [values, setValues] = useState<AIConnectionFormValues>({
    provider: initial?.provider ?? 'openai', apiKey: '', baseUrl: initial?.base_url ?? '', model: initial?.model ?? '',
    fallbackModels: initial?.fallback_models ?? '', label: initial?.label ?? '',
  })
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const set = (field: keyof AIConnectionFormValues) => (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>) =>
    setValues(v => ({ ...v, [field]: e.target.value }))

  const { provider } = values
  const selfHosted = SELF_HOSTED.has(provider)
  const keepsKey = !!initial?.has_api_key

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setSaving(true)
    setError('')
    try {
      await onSave(values)
    } catch (err) {
      setError(apiErrorText(err) ?? t('ai.connectFailed'))
    } finally {
      setSaving(false)
    }
  }

  return (
    <Dialog onClose={onClose} className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 dark:bg-black/60 bg-black/30" onClick={onClose} />
      <div className="relative w-full max-w-sm bg-raised rounded-xl shadow-2xl border border-line p-5">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-sm font-semibold text-ink">{initial ? t('ai.editTitle') : t('ai.addTitle')}</h3>
          <button onClick={onClose} aria-label={t('common:actions.close')}>
            <X size={14} className="text-ink-4" />
          </button>
        </div>
        <form onSubmit={handleSubmit} className="space-y-3">
          <div>
            <label htmlFor="ai-provider" className={LABEL}>{t('ai.provider')}</label>
            <select id="ai-provider" value={provider} onChange={set('provider')} className={INPUT}>
              {AI_PROVIDERS.map(p => <option key={p} value={p}>{providerLabel(t, p)}</option>)}
            </select>
          </div>
          <div>
            <label htmlFor="ai-label" className={LABEL}>{t('ai.label')}</label>
            <input id="ai-label" type="text" maxLength={80} value={values.label} onChange={set('label')}
              placeholder={t('ai.labelPlaceholder')} className={INPUT} />
          </div>
          {!selfHosted && (
            <div>
              <label htmlFor="ai-api-key" className={LABEL}>{t('ai.apiKey')}</label>
              <input id="ai-api-key" type="password" required={!keepsKey} autoFocus={!initial} value={values.apiKey}
                onChange={set('apiKey')}
                placeholder={keepsKey ? t('ai.keySaved') : (API_KEY_PLACEHOLDER[provider] ?? 'sk-...')}
                className={`${INPUT} font-mono`} />
            </div>
          )}
          <div>
            <label htmlFor="ai-base-url" className={LABEL}>{selfHosted ? t('ai.serverUrl') : t('ai.baseUrl')}</label>
            <input id="ai-base-url" type="url" required={selfHosted} value={values.baseUrl} onChange={set('baseUrl')}
              placeholder={selfHosted ? 'http://localhost:11434' : 'https://...'} className={INPUT} />
          </div>
          <div>
            <label htmlFor="ai-model" className={LABEL}>{t('ai.model')}</label>
            <input id="ai-model" type="text" value={values.model} onChange={set('model')}
              placeholder={DEFAULT_MODEL[provider] ?? t('ai.modelPlaceholder')} className={INPUT} />
          </div>
          {PROVIDERS_WITH_FALLBACK.has(provider) && (
            <div>
              <label htmlFor="ai-fallback-models" className={LABEL}>{t('ai.fallbackModels')}</label>
              <input id="ai-fallback-models" type="text" value={values.fallbackModels} onChange={set('fallbackModels')}
                placeholder={FALLBACK_PLACEHOLDER[provider]} className={INPUT} />
              <p className="mt-1.5 text-2xs text-ink-4">{t('ai.fallbackHelp')}</p>
            </div>
          )}
          {error && <p className="text-xs text-danger bg-red-50 dark:bg-red-500/10 rounded-lg px-2 py-1.5">{error}</p>}
          <button type="submit" disabled={saving}
            className="w-full py-2 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover disabled:opacity-50 transition-colors">
            {saving ? t('common:actions.saving') : t('common:actions.save')}
          </button>
        </form>
      </div>
    </Dialog>
  )
}
