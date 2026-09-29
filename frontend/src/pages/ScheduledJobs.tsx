import { useCallback, useEffect, useState } from 'react'
import { Clock, ShieldAlert } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useAuth } from '../auth/AuthContext'
import * as api from '../scheduler/api'
import type { Job, SyncScheduleRow } from '../scheduler/types'
import { useDateFormat } from '../format/DateFormatProvider'

function StatusBadge({ status }: { status: string }) {
  const { t } = useTranslation('settings')
  if (status === 'success') return <span className="text-success">{t('jobs.statusSuccess')}</span>
  if (status === 'error') return <span className="text-danger">{t('jobs.statusError')}</span>
  return <span className="text-ink-4">{t('jobs.statusNeverRun')}</span>
}

// ScheduledJobs is the super_admin "Scheduled Jobs" admin page -- global
// cron jobs (one run touches every tenant) and every tenant's WooCommerce
// sync schedules, cross-tenant, each with a Run Now action. Gated to
// super_admin here (the backend API is gated identically via
// middleware.RequireRole) since a "run now" on a global job would trigger
// it for every tenant at once, not just whoever's viewing the page.
interface JobConfigEdit {
  enabled: boolean
  interval_minutes: string
}

export default function ScheduledJobs() {
  const { formatDateTime } = useDateFormat()
  const { t } = useTranslation(['settings', 'common'])
  const { token, user } = useAuth()
  const [jobs, setJobs] = useState<Job[]>([])
  const [schedules, setSchedules] = useState<SyncScheduleRow[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [runningKey, setRunningKey] = useState<string | null>(null)
  const [edits, setEdits] = useState<Record<string, JobConfigEdit>>({})
  const [savingKey, setSavingKey] = useState<string | null>(null)

  const isSuperAdmin = user?.role === 'super_admin'

  const load = useCallback(async () => {
    if (!isSuperAdmin) return
    setLoading(true)
    setError('')
    try {
      const [jobsResult, schedulesResult] = await Promise.all([
        api.listJobs(token ?? ''),
        api.listSyncSchedules(token ?? ''),
      ])
      setJobs(jobsResult)
      setSchedules(schedulesResult)
      setEdits(Object.fromEntries(jobsResult.map(j => [j.job_key, { enabled: j.enabled, interval_minutes: String(j.interval_minutes) }])))
    } catch (err) {
      setError(err instanceof Error ? err.message : t('jobs.loadFailed'))
    } finally {
      setLoading(false)
    }
  }, [token, isSuperAdmin, t])

  useEffect(() => { load() }, [load])

  async function handleRunJobNow(jobKey: string) {
    setRunningKey(jobKey)
    setError('')
    try {
      await api.runJobNow(token ?? '', jobKey)
      await load()
    } catch (err) {
      setError(err instanceof Error ? err.message : t('jobs.runFailed'))
    } finally {
      setRunningKey(null)
    }
  }

  async function handleRunScheduleNow(id: string) {
    setRunningKey(id)
    setError('')
    try {
      await api.runSyncScheduleNow(token ?? '', id)
      await load()
    } catch (err) {
      setError(err instanceof Error ? err.message : t('jobs.triggerFailed'))
    } finally {
      setRunningKey(null)
    }
  }

  function isDirty(j: Job): boolean {
    const edit = edits[j.job_key]
    if (!edit) return false
    return edit.enabled !== j.enabled || edit.interval_minutes !== String(j.interval_minutes)
  }

  async function handleSaveJobConfig(j: Job) {
    const edit = edits[j.job_key]
    const intervalMinutes = parseInt(edit.interval_minutes, 10)
    if (!Number.isFinite(intervalMinutes) || intervalMinutes < 1) {
      setError(t('jobs.intervalMin'))
      return
    }
    setSavingKey(j.job_key)
    setError('')
    try {
      await api.updateJobConfig(token ?? '', j.job_key, edit.enabled, intervalMinutes)
      await load()
    } catch (err) {
      setError(err instanceof Error ? err.message : t('jobs.saveFailed'))
    } finally {
      setSavingKey(null)
    }
  }

  if (!isSuperAdmin) {
    return (
      <div className="flex flex-col items-center justify-center h-full gap-2 text-ink-4">
        <ShieldAlert size={24} />
        <p className="text-sm">{t('jobs.superAdminOnly')}</p>
      </div>
    )
  }

  return (
    <div className="flex flex-col h-full overflow-auto">
      <div className="flex items-center gap-2 px-5 py-3 border-b border-line bg-sunken shrink-0">
        <Clock size={14} className="text-accent" />
        <h1 className="text-sm font-semibold text-ink">{t('jobs.title')}</h1>
      </div>

      {error && <div className="px-5 py-2 text-xs text-danger">{error}</div>}
      {loading && <div className="px-5 py-6 text-xs text-center text-ink-4">{t('common:status.loading')}</div>}

      {!loading && (
        <div className="p-5 flex flex-col gap-8">
          <section>
            <h2 className="text-xs font-semibold uppercase tracking-wider text-ink-3 mb-2">
              {t('jobs.global', { count: jobs.length })}
            </h2>
            <table className="w-full text-xs">
              <thead>
                <tr className="text-ink-4 text-left">
                  <th className="font-medium pb-2">{t('jobs.job')}</th>
                  <th className="font-medium pb-2">{t('common:status.enabled')}</th>
                  <th className="font-medium pb-2">{t('jobs.intervalMinutes')}</th>
                  <th className="font-medium pb-2">{t('jobs.lastRun')}</th>
                  <th className="font-medium pb-2">{t('common:fields.status')}</th>
                  <th className="font-medium pb-2">{t('jobs.nextRun')}</th>
                  <th className="pb-2" />
                </tr>
              </thead>
              <tbody className="divide-y divide-line/60">
                {jobs.map(j => {
                  const edit = edits[j.job_key] ?? { enabled: j.enabled, interval_minutes: String(j.interval_minutes) }
                  const dirty = isDirty(j)
                  return (
                    <tr key={j.job_key}>
                      <td className="py-2 text-ink">{j.job_key}</td>
                      <td className="py-2">
                        <input
                          type="checkbox"
                          checked={edit.enabled}
                          onChange={e => setEdits(prev => ({ ...prev, [j.job_key]: { ...edit, enabled: e.target.checked } }))}
                          aria-label={t('jobs.enabledFor', { job: j.job_key })}
                        />
                      </td>
                      <td className="py-2">
                        <input
                          type="number"
                          min={1}
                          value={edit.interval_minutes}
                          onChange={e => setEdits(prev => ({ ...prev, [j.job_key]: { ...edit, interval_minutes: e.target.value } }))}
                          aria-label={t('jobs.intervalFor', { job: j.job_key })}
                          className="w-20 px-2 py-1 rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus"
                        />
                      </td>
                      <td className="py-2 text-ink-3">{formatDateTime(j.last_run_at)}</td>
                      <td className="py-2"><StatusBadge status={j.last_run_status} /></td>
                      <td className="py-2 text-ink-3">{formatDateTime(j.next_run_at)}</td>
                      <td className="py-2 text-right space-x-2 whitespace-nowrap">
                        {dirty && (
                          <button
                            onClick={() => handleSaveJobConfig(j)}
                            disabled={savingKey === j.job_key}
                            className="px-3 py-1.5 rounded-lg bg-fill text-2xs font-medium text-ink-2 hover:bg-hover disabled:opacity-50 transition-colors"
                          >
                            {savingKey === j.job_key ? t('common:actions.saving') : t('common:actions.save')}
                          </button>
                        )}
                        <button
                          onClick={() => handleRunJobNow(j.job_key)}
                          disabled={runningKey === j.job_key}
                          className="px-3 py-1.5 rounded-lg bg-accent-solid text-white text-2xs font-medium hover:bg-accent-solid-hover disabled:opacity-50 transition-colors"
                        >
                          {runningKey === j.job_key ? t('jobs.running') : t('jobs.runNow')}
                        </button>
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </section>

          <section>
            <h2 className="text-xs font-semibold uppercase tracking-wider text-ink-3 mb-2">
              {t('jobs.schedules', { count: schedules.length })}
            </h2>
            <table className="w-full text-xs">
              <thead>
                <tr className="text-ink-4 text-left">
                  <th className="font-medium pb-2">{t('jobs.store')}</th>
                  <th className="font-medium pb-2">{t('jobs.entity')}</th>
                  <th className="font-medium pb-2">{t('jobs.direction')}</th>
                  <th className="font-medium pb-2">{t('jobs.interval')}</th>
                  <th className="font-medium pb-2">{t('jobs.lastRun')}</th>
                  <th className="font-medium pb-2">{t('common:fields.status')}</th>
                  <th className="pb-2" />
                </tr>
              </thead>
              <tbody className="divide-y divide-line/60">
                {schedules.length === 0 && (
                  <tr><td colSpan={7} className="py-4 text-center text-ink-4">{t('jobs.noSchedules')}</td></tr>
                )}
                {schedules.map(s => (
                  <tr key={s.id}>
                    <td className="py-2 text-ink">{s.connection_store_url || s.connection_id}</td>
                    <td className="py-2 text-ink-3">{s.entity_type}</td>
                    <td className="py-2 text-ink-3">{s.direction}</td>
                    <td className="py-2 text-ink-3">{t('jobs.minutesShort', { minutes: s.interval_minutes })}</td>
                    <td className="py-2 text-ink-3">{formatDateTime(s.last_run_at)}</td>
                    <td className="py-2"><StatusBadge status={s.last_run_status} /></td>
                    <td className="py-2 text-right">
                      <button
                        onClick={() => handleRunScheduleNow(s.id)}
                        disabled={runningKey === s.id || !s.enabled}
                        title={!s.enabled ? t('jobs.scheduleDisabled') : undefined}
                        className="px-3 py-1.5 rounded-lg bg-accent-solid text-white text-2xs font-medium hover:bg-accent-solid-hover disabled:opacity-50 transition-colors"
                      >
                        {runningKey === s.id ? t('jobs.running') : t('jobs.runNow')}
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </section>
        </div>
      )}
    </div>
  )
}
