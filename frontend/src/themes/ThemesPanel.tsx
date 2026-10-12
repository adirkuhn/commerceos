import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { ArrowLeft, ExternalLink, History, Paintbrush, Plus } from 'lucide-react'
import { useAuth } from '../auth/AuthContext'
import { useConfirm } from '../components/common/useConfirm'
import { useDateFormat } from '../format/DateFormatProvider'
import { useQueryString } from '../routing/useQueryState'
import type { SalesChannel } from '../integrations/types'
import { createThemeSession } from './api'
import { editorURL, storeURL } from './storefrontURL'
import { loadThemePreference } from '../theme/themePreference'
import { currentLocale } from '../i18n/i18n'
import { useThemes } from './useThemes'
import { VersionHistory } from './VersionHistory'
import type { Theme, ThemePreset, VersionSummary } from './types'
import { apiErrorText } from '../api/apiErrorText'

interface Props {
  channel: SalesChannel
  onClose: () => void
}

// A channel's theme library: create themes, open one in the visual editor
// (hosted by the storefront app), delete unused ones. The live theme is
// what shoppers see at /store/{channel}.
function localStorageOrNull(): Storage | null {
  try {
    return window.localStorage
  } catch {
    return null
  }
}

export function ThemesPanel({ channel, onClose }: Props) {
  const { token } = useAuth()
  const { t } = useTranslation(['themes', 'common'])
  const { themes, loading, error, create, remove, makeLive } = useThemes(token ?? '', channel.id)
  const [historyFor, setHistoryFor] = useQueryString('history')
  const { confirm, confirmDialog } = useConfirm()
  const { formatDate } = useDateFormat()
  const [name, setName] = useState('')
  const [preset, setPreset] = useState<ThemePreset>('default')
  const [creating, setCreating] = useState(false)
  const [actionError, setActionError] = useState('')
  const hasLive = themes.some(t => t.is_live)

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!name.trim()) return
    setCreating(true)
    setActionError('')
    try {
      await create(name.trim(), preset)
      setName('')
    } catch (err) {
      setActionError(apiErrorText(err) ?? t('errors.create'))
    } finally {
      setCreating(false)
    }
  }

  const customize = async (theme: Theme) => {
    setActionError('')
    // Open the tab now, inside the click, or the browser blocks the popup.
    const win = window.open('', '_blank')
    try {
      const session = await createThemeSession(token ?? '', theme.id, 'edit', currentLocale())
      if (win) {
        win.opener = null
        win.location.href = editorURL(session.token, undefined, loadThemePreference(localStorageOrNull()))
      }
    } catch (err) {
      win?.close()
      setActionError(apiErrorText(err) ?? t('errors.openEditor'))
    }
  }

  const handleDelete = async (theme: Theme) => {
    const ok = await confirm({
      title: t('delete.title', { name: theme.name }),
      message: t('delete.message'),
      confirmLabel: t('delete.confirm'),
      destructive: true,
    })
    if (!ok) return
    setActionError('')
    try {
      await remove(theme.id)
    } catch (err) {
      setActionError(apiErrorText(err) ?? t('errors.delete'))
    }
  }

  const handleMakeLive = async (theme: Theme, version: VersionSummary) => {
    setActionError('')
    try {
      await makeLive(theme.id, version.id)
    } catch (err) {
      setActionError(apiErrorText(err) ?? t('errors.makeLive'))
    }
  }

  return (
    <div className="flex-1 flex flex-col overflow-hidden p-5 gap-4">
      <div className="flex items-center gap-3">
        <button onClick={onClose} aria-label={t('backToChannels')} className="p-1.5 rounded-lg bg-fill text-ink-3 hover:text-accent">
          <ArrowLeft size={14} />
        </button>
        <h2 className="text-sm font-semibold text-ink">{t('title', { channel: channel.name })}</h2>
        {hasLive && (
          <a href={storeURL(channel.id)} target="_blank" rel="noopener noreferrer"
            className="ml-auto flex items-center gap-1 text-2xs text-accent">
            {t('viewStore')} <ExternalLink size={11} />
          </a>
        )}
      </div>

      {channel.storefront_mode !== 'theme' && (
        <p className="text-xs text-warning bg-amber-500/10 rounded-lg px-3 py-2">
          {t('headlessWarning')}
        </p>
      )}

      <form onSubmit={handleCreate} className="flex flex-wrap gap-2 max-w-xl">
        <label htmlFor="new-theme-name" className="sr-only">{t('newThemeName')}</label>
        <input id="new-theme-name" value={name} onChange={e => setName(e.target.value)} maxLength={120}
          placeholder={t('newThemeName')}
          className="flex-1 px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus" />
        <label htmlFor="new-theme-preset" className="sr-only">{t('preset.label')}</label>
        <select id="new-theme-preset" value={preset} onChange={e => setPreset(e.target.value as ThemePreset)}
          className="px-2 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus">
          <option value="default">{t('preset.default')}</option>
          <option value="mercado">{t('preset.mercado')}</option>
          <option value="fresh">{t('preset.fresh')}</option>
        </select>
        <button type="submit" disabled={creating || !name.trim() || channel.storefront_mode !== 'theme'}
          className="flex items-center gap-1 px-3 py-2 rounded-lg bg-accent-solid text-white text-xs font-medium disabled:opacity-50">
          <Plus size={12} /> {creating ? t('creating') : t('create')}
        </button>
      </form>

      {(error || actionError) && <p className="text-xs text-danger" role="alert">{actionError || error}</p>}

      {loading ? (
        <p className="text-xs text-ink-4">{t('loading')}</p>
      ) : themes.length === 0 ? (
        <p className="text-xs text-ink-4">{t('empty')}</p>
      ) : (
        <ul className="flex flex-col gap-2 overflow-y-auto" aria-label={t('listLabel')}>
          {themes.map(theme => (
            <li key={theme.id} className="rounded-xl border border-line bg-raised">
              <div className="flex items-center gap-3 p-3">
                <div className="min-w-0 flex-1">
                  <p className="text-sm font-medium text-ink truncate">{theme.name}</p>
                  <p className="text-2xs text-ink-4">{t('updated', { date: formatDate(theme.updated_at) })}</p>
                </div>
                {theme.is_live
                  ? <span className="text-2xs px-2 py-0.5 rounded-full bg-green-500/10 text-green-700 dark:text-green-400">{t('live')}</span>
                  : <span className="text-2xs px-2 py-0.5 rounded-full bg-fill text-ink-3">{theme.published_version_id ? t('notLive') : t('draftOnly')}</span>}
                <button onClick={() => void customize(theme)} disabled={channel.storefront_mode !== 'theme'}
                  className="flex items-center gap-1 px-3 py-1.5 rounded-lg bg-fill text-2xs text-ink-2 hover:text-accent disabled:opacity-50">
                  <Paintbrush size={11} /> {t('customize')}
                </button>
                <button onClick={() => void handleDelete(theme)} disabled={theme.is_live}
                  title={theme.is_live ? t('liveCannotDelete') : undefined}
                  className="px-3 py-1.5 rounded-lg bg-fill text-2xs text-ink-3 hover:text-danger disabled:opacity-40">
                  {t('common:actions.delete')}
                </button>
                {theme.published_version_id && (
                  <button onClick={() => setHistoryFor(historyFor === theme.id ? null : theme.id)} aria-expanded={historyFor === theme.id}
                    className="flex items-center gap-1 px-3 py-1.5 rounded-lg bg-fill text-2xs text-ink-2 hover:text-accent">
                    <History size={11} /> {t('versions')}
                  </button>
                )}
              </div>
              {historyFor === theme.id && <VersionHistory token={token ?? ''} theme={theme} onMakeLive={handleMakeLive} />}
            </li>
          ))}
        </ul>
      )}
      {confirmDialog}
    </div>
  )
}
