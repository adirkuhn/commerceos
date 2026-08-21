import { useCallback, useEffect, useState } from 'react'
import { Clock, ShieldAlert } from 'lucide-react'
import { useAuth } from '../auth/AuthContext'
import * as api from '../scheduler/api'
import type { Job, SyncScheduleRow } from '../scheduler/types'

function formatDate(d: string | null): string {
  return d ? new Date(d).toLocaleString() : '—'
}

function StatusBadge({ status }: { status: string }) {
  if (status === 'success') return <span className="text-green-500">success</span>
  if (status === 'error') return <span className="text-red-400">error</span>
  return <span className="dark:text-gray-500 text-gray-400">never run</span>
}

// ScheduledJobs is the super_admin "Scheduled Jobs" admin page -- global
// cron jobs (one run touches every tenant) and every tenant's WooCommerce
// sync schedules, cross-tenant, each with a Run Now action. Gated to
// super_admin here (the backend API is gated identically via
// middleware.RequireRole) since a "run now" on a global job would trigger
// it for every tenant at once, not just whoever's viewing the page.
export default function ScheduledJobs() {
  const { token, user } = useAuth()
  const [jobs, setJobs] = useState<Job[]>([])
  const [schedules, setSchedules] = useState<SyncScheduleRow[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [runningKey, setRunningKey] = useState<string | null>(null)

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
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load scheduled jobs')
    } finally {
      setLoading(false)
    }
  }, [token, isSuperAdmin])

  useEffect(() => { load() }, [load])

  async function handleRunJobNow(jobKey: string) {
    setRunningKey(jobKey)
    setError('')
    try {
      await api.runJobNow(token ?? '', jobKey)
      await load()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to run job')
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
      setError(err instanceof Error ? err.message : 'Failed to trigger sync schedule')
    } finally {
      setRunningKey(null)
    }
  }

  if (!isSuperAdmin) {
    return (
      <div className="flex flex-col items-center justify-center h-full gap-2 dark:text-gray-500 text-gray-400">
        <ShieldAlert size={24} />
        <p className="text-sm">Super admin access required.</p>
      </div>
    )
  }

  return (
    <div className="flex flex-col h-full overflow-auto">
      <div className="flex items-center gap-2 px-5 py-3 border-b dark:border-[#2D3748] border-gray-200 dark:bg-[#111827] bg-gray-50 shrink-0">
        <Clock size={14} className="text-blue-400" />
        <h1 className="text-sm font-semibold dark:text-gray-100 text-gray-900">Scheduled Jobs</h1>
      </div>

      {error && <div className="px-5 py-2 text-xs text-red-400">{error}</div>}
      {loading && <div className="px-5 py-6 text-xs text-center dark:text-gray-500 text-gray-400">Loading…</div>}

      {!loading && (
        <div className="p-5 flex flex-col gap-8">
          <section>
            <h2 className="text-xs font-semibold uppercase tracking-wider dark:text-gray-400 text-gray-500 mb-2">
              Global jobs ({jobs.length})
            </h2>
            <table className="w-full text-xs">
              <thead>
                <tr className="dark:text-gray-500 text-gray-400 text-left">
                  <th className="font-medium pb-2">Job</th>
                  <th className="font-medium pb-2">Interval</th>
                  <th className="font-medium pb-2">Last run</th>
                  <th className="font-medium pb-2">Status</th>
                  <th className="font-medium pb-2">Next run</th>
                  <th className="pb-2" />
                </tr>
              </thead>
              <tbody className="divide-y dark:divide-[#2D3748]/60 divide-gray-100">
                {jobs.map(j => (
                  <tr key={j.job_key}>
                    <td className="py-2 dark:text-gray-100 text-gray-900">{j.job_key}</td>
                    <td className="py-2 dark:text-gray-400 text-gray-500">{j.interval_minutes}m</td>
                    <td className="py-2 dark:text-gray-400 text-gray-500">{formatDate(j.last_run_at)}</td>
                    <td className="py-2"><StatusBadge status={j.last_run_status} /></td>
                    <td className="py-2 dark:text-gray-400 text-gray-500">{formatDate(j.next_run_at)}</td>
                    <td className="py-2 text-right">
                      <button
                        onClick={() => handleRunJobNow(j.job_key)}
                        disabled={runningKey === j.job_key}
                        className="px-3 py-1.5 rounded-lg bg-blue-600 text-white text-[11px] font-medium hover:bg-blue-700 disabled:opacity-50 transition-colors"
                      >
                        {runningKey === j.job_key ? 'Running…' : 'Run now'}
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </section>

          <section>
            <h2 className="text-xs font-semibold uppercase tracking-wider dark:text-gray-400 text-gray-500 mb-2">
              WooCommerce sync schedules ({schedules.length})
            </h2>
            <table className="w-full text-xs">
              <thead>
                <tr className="dark:text-gray-500 text-gray-400 text-left">
                  <th className="font-medium pb-2">Store</th>
                  <th className="font-medium pb-2">Entity</th>
                  <th className="font-medium pb-2">Direction</th>
                  <th className="font-medium pb-2">Interval</th>
                  <th className="font-medium pb-2">Last run</th>
                  <th className="font-medium pb-2">Status</th>
                  <th className="pb-2" />
                </tr>
              </thead>
              <tbody className="divide-y dark:divide-[#2D3748]/60 divide-gray-100">
                {schedules.length === 0 && (
                  <tr><td colSpan={7} className="py-4 text-center dark:text-gray-500 text-gray-400">No sync schedules configured</td></tr>
                )}
                {schedules.map(s => (
                  <tr key={s.id}>
                    <td className="py-2 dark:text-gray-100 text-gray-900">{s.connection_store_url || s.connection_id}</td>
                    <td className="py-2 dark:text-gray-400 text-gray-500">{s.entity_type}</td>
                    <td className="py-2 dark:text-gray-400 text-gray-500">{s.direction}</td>
                    <td className="py-2 dark:text-gray-400 text-gray-500">{s.interval_minutes}m</td>
                    <td className="py-2 dark:text-gray-400 text-gray-500">{formatDate(s.last_run_at)}</td>
                    <td className="py-2"><StatusBadge status={s.last_run_status} /></td>
                    <td className="py-2 text-right">
                      <button
                        onClick={() => handleRunScheduleNow(s.id)}
                        disabled={runningKey === s.id || !s.enabled}
                        title={!s.enabled ? 'Schedule is disabled' : undefined}
                        className="px-3 py-1.5 rounded-lg bg-blue-600 text-white text-[11px] font-medium hover:bg-blue-700 disabled:opacity-50 transition-colors"
                      >
                        {runningKey === s.id ? 'Running…' : 'Run now'}
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
