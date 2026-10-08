import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { GitCompare } from 'lucide-react'
import Pagination from '../components/common/Pagination'
import { useQueryPage } from '../routing/useQueryState'
import { useDateFormat } from '../format/DateFormatProvider'
import { formatMoney } from '../format/money'
import { useConnections } from '../integrations/hooks/useConnections'
import { useOrderDifferences, ORDER_DIFFERENCES_PAGE_SIZE } from './hooks/useOrderDifferences'
import { useWooOrderAudit } from './hooks/useWooOrderAudit'
import { describeDifference } from './orderDifferences'
import type { OrderDifference } from './orderDifferences'

const WOO_PROVIDER = 'woocommerce'

// Where imported orders differ from the channel's own order (e.g. an item
// in the Woo order the CommerceOS order doesn't have). Nothing here
// changes an order: staff fix it by hand (the difference then resolves
// itself on the next comparison) or dismiss it.
export function OrderDifferencesSection({ token }: { token: string }) {
  const { t } = useTranslation(['stockIssues', 'common'])
  const { formatDateTime } = useDateFormat()
  // Its own page key so it doesn't fight the stock list's ?page=.
  const [page, setPage] = useQueryPage('differences_page')
  const { differences, total, loading, error, refresh, dismiss } = useOrderDifferences(token, page)
  const { connections } = useConnections(token)
  const wooConnections = connections.filter(c => c.provider === WOO_PROVIDER)
  const audit = useWooOrderAudit(token, refresh)
  const [dismissingId, setDismissingId] = useState<string | null>(null)
  const [dismissError, setDismissError] = useState('')

  async function handleDismiss(id: string) {
    setDismissingId(id)
    setDismissError('')
    try {
      await dismiss(id)
    } catch (err) {
      setDismissError(err instanceof Error ? err.message : t('differences.dismissFailed'))
    } finally {
      setDismissingId(null)
    }
  }

  function describe(d: OrderDifference): string {
    const { key, values } = describeDifference(d)
    if (key === 'total') {
      return t('differences.kind.total', {
        channelTotal: formatMoney(d.channel_total_minor_units, d.currency_code),
        localTotal: formatMoney(d.local_total_minor_units, d.currency_code),
      })
    }
    return t(`differences.kind.${key}`, values)
  }

  return (
    <section className="border-t border-line">
      <div className="flex flex-wrap items-center gap-3 px-5 py-3 bg-sunken border-b border-line">
        <GitCompare size={14} className="text-warning" />
        <h2 className="text-sm font-semibold text-ink">{t('differences.title')}</h2>
        <span className="text-xs text-ink-4">{t('differences.count', { count: total })}</span>
        <div className="flex items-center gap-2 ml-auto">
          {wooConnections.map(c => (
            <button
              key={c.id} type="button" onClick={() => audit.run(c.id)} disabled={audit.running}
              className="px-3 py-1.5 rounded-lg border border-line text-ink-2 text-2xs font-medium hover:bg-hover disabled:opacity-50 transition-colors"
            >
              {t('differences.audit', { store: c.store_url })}
            </button>
          ))}
        </div>
      </div>

      {(audit.running || audit.done || audit.error) && (
        <div className="px-5 py-2 text-xs border-b border-line space-y-1">
          {audit.running && <p className="text-ink-3">{t('differences.auditing', { count: audit.checked })}</p>}
          {audit.done && <p className="text-ink-3">{t('differences.audited', { count: audit.checked })}</p>}
          {audit.error && <p className="text-danger">{audit.error}</p>}
          {audit.failures.map(f => <p key={f} className="text-warning">{f}</p>)}
        </div>
      )}
      {(error || dismissError) && <p className="px-5 py-2 text-xs text-danger">{error || dismissError}</p>}

      <table className="w-full text-xs">
        <thead className="bg-sunken border-b border-line">
          <tr>
            <th className="px-3 py-2.5 text-left font-medium uppercase tracking-wider text-ink-4">{t('colOrder')}</th>
            <th className="px-3 py-2.5 text-left font-medium uppercase tracking-wider text-ink-4">{t('colCustomer')}</th>
            <th className="px-3 py-2.5 text-left font-medium uppercase tracking-wider text-ink-4">{t('differences.colDifference')}</th>
            <th className="px-3 py-2.5 w-28" />
          </tr>
        </thead>
        <tbody className="divide-y divide-line/60">
          {loading && differences.length === 0 && (
            <tr><td colSpan={4} className="px-4 py-6 text-center text-ink-4">{t('common:status.loading')}</td></tr>
          )}
          {!loading && differences.length === 0 && (
            <tr><td colSpan={4} className="px-4 py-6 text-center text-ink-4">{t('differences.empty')}</td></tr>
          )}
          {differences.map(d => (
            <tr key={d.id}>
              <td className="px-3 py-2.5">
                <Link to={`/sales-orders/${d.sales_order_id}`} className="font-medium text-accent hover:underline">#{d.order_number}</Link>
                <p className="text-2xs text-ink-4">{formatDateTime(d.order_occurred_at)}</p>
              </td>
              <td className="px-3 py-2.5 text-ink-3">{d.customer_name}</td>
              <td className="px-3 py-2.5 text-ink">{describe(d)}</td>
              <td className="px-3 py-2.5 text-right">
                <button
                  type="button" onClick={() => handleDismiss(d.id)} disabled={dismissingId === d.id}
                  className="px-3 py-1.5 rounded-lg border border-line text-ink-2 text-2xs font-medium hover:bg-hover disabled:opacity-50 transition-colors"
                >
                  {t('differences.dismiss')}
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>

      <Pagination page={page} pageSize={ORDER_DIFFERENCES_PAGE_SIZE} totalItems={total} onPageChange={setPage} />
    </section>
  )
}
