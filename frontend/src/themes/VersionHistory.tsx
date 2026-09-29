import { useTranslation } from 'react-i18next'
import { useConfirm } from '../components/common/useConfirm'
import { useDateFormat } from '../format/DateFormatProvider'
import { useResource } from '../routing/useResource'
import { listVersions } from './api'
import type { Theme, VersionSummary } from './types'

interface Props {
  token: string
  theme: Theme
  onMakeLive: (theme: Theme, version: VersionSummary) => Promise<void>
}

/** A theme's published versions, newest first; any of them can be made
 *  live again (a rollback -- the draft is left as it is). */
export function VersionHistory({ token, theme, onMakeLive }: Props) {
  const { formatDateTime } = useDateFormat()
  const { t } = useTranslation('themes')
  const { confirm, confirmDialog } = useConfirm()
  const { data, loading, error } = useResource(() => listVersions(token, theme.id), [token, theme.id, theme.published_version_id, theme.is_live])
  const published = (data ?? []).filter(v => v.state === 'sealed')

  const makeLive = async (v: VersionSummary) => {
    const ok = await confirm({
      title: t('history.confirmTitle', { number: v.number, name: theme.name }),
      message: t('history.confirmMessage'),
      confirmLabel: t('history.makeLive'),
    })
    if (ok) await onMakeLive(theme, v)
  }

  if (loading) return <p className="text-2xs text-ink-4 px-3 pb-3">{t('history.loading')}</p>
  if (error) return <p role="alert" className="text-2xs text-danger px-3 pb-3">{error}</p>
  if (!published.length) return <p className="text-2xs text-ink-4 px-3 pb-3">{t('history.notPublished')}</p>
  return (
    <>
      <ol aria-label={t('history.listLabel', { name: theme.name })} className="flex flex-col divide-y divide-line/60 border-t border-line">
        {published.map(v => {
          const current = theme.published_version_id === v.id
          const live = current && theme.is_live
          return (
            <li key={v.id} className="flex items-center gap-3 px-3 py-2">
              <span className="text-xs font-medium text-ink">{t('history.version', { number: v.number })}</span>
              <span className="text-2xs text-ink-4">{t('history.published', { date: formatDateTime(v.sealed_at ?? v.created_at) })}</span>
              {v.note && <span className="text-2xs text-ink-3 truncate">{v.note}</span>}
              {live
                ? <span className="ml-auto text-2xs px-2 py-0.5 rounded-full bg-green-500/10 text-green-700 dark:text-green-400">{t('live')}</span>
                : (
                  <button onClick={() => void makeLive(v)}
                    className="ml-auto px-3 py-1 rounded-lg bg-fill text-2xs text-ink-2 hover:text-accent">
                    {t('history.makeLive')}
                  </button>
                )}
            </li>
          )
        })}
      </ol>
      {confirmDialog}
    </>
  )
}
