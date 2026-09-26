import { ArrowLeft } from 'lucide-react'
import clsx from 'clsx'
import { useQueryEnum } from '../routing/useQueryState'
import { useBookingDiscrepancies } from './hooks/useBookingDiscrepancies'
import { useDateFormat } from '../format/DateFormatProvider'

const DISCREPANCY_TABS = ['missing', 'damaged-lost'] as const

type DiscrepancyTab = typeof DISCREPANCY_TABS[number]

// Tenant-wide, derived on demand (not persisted) from every completed
// booking's line items vs scans -- same "not a real table" reasoning as
// the Inventory module's Expiry & Discards report.
export function BookingDiscrepanciesPage({ token, onClose }: { token: string; onClose: () => void }) {
  const { formatDate } = useDateFormat()
  const { report, loading, error } = useBookingDiscrepancies(token)
  const [tab, setTab] = useQueryEnum('tab', DISCREPANCY_TABS, 'missing')

  const TABS: { id: DiscrepancyTab; label: string }[] = [
    { id: 'missing', label: `Missing${report.missing_items.length ? ` (${report.missing_items.length})` : ''}` },
    { id: 'damaged-lost', label: `Damaged / Lost${report.damaged_lost.length ? ` (${report.damaged_lost.length})` : ''}` },
  ]

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-6 py-4 border-b border-line bg-sunken shrink-0">
        <div>
          <button
            onClick={onClose}
            className="flex items-center gap-1 text-2xs font-medium text-ink-4 hover:text-ink-2 transition-colors mb-1"
          >
            <ArrowLeft size={11} /> Procurement
          </button>
          <h1 className="text-xl font-semibold text-ink leading-tight">Booking discrepancies</h1>
          <p className="text-xs text-ink-4 mt-0.5">Every completed booking's under-scanned line items and damaged/lost stock</p>
        </div>
      </div>

      <div className="flex gap-2 px-6 pt-4 shrink-0">
        {TABS.map(t => (
          <button aria-pressed={tab === t.id}
            key={t.id}
            onClick={() => setTab(t.id)}
            className={clsx(
              'px-3 py-1.5 rounded-lg text-xs font-medium transition-colors',
              tab === t.id
                ? 'bg-accent-solid text-white'
                : 'bg-raised text-ink-3 hover:bg-hover',
            )}
          >
            {t.label}
          </button>
        ))}
      </div>

      <div className="flex-1 overflow-auto">
        {error && <p className="px-6 py-4 text-xs text-danger">{error}</p>}

        {tab === 'missing' && (
          <>
            {!error && loading && report.missing_items.length === 0 && <p className="px-6 py-4 text-xs text-ink-4">Loading…</p>}
            {!error && !loading && report.missing_items.length === 0 && (
              <p className="px-6 py-4 text-xs text-ink-4">No missing items -- every completed booking was fully scanned.</p>
            )}
            {report.missing_items.length > 0 && (
              <table className="w-full text-xs">
                <thead className="sticky top-0 z-10 bg-sunken border-b border-line">
                  <tr>
                    {['SKU', 'Product', 'Expected', 'Scanned', 'Missing', 'Completed'].map(h => (
                      <th key={h} className="px-4 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">{h}</th>
                    ))}
                  </tr>
                </thead>
                <tbody className="divide-y divide-line/60">
                  {report.missing_items.map(m => (
                    <tr key={m.line_item_id} className="hover:bg-hover/60 transition-colors">
                      <td className="px-4 py-2.5 font-mono text-ink-2">{m.sku_code ?? '—'}</td>
                      <td className="px-4 py-2.5 text-ink">{m.product_name ?? m.raw_description}</td>
                      <td className="px-4 py-2.5 tabular-nums text-ink-2">{m.expected_qty}</td>
                      <td className="px-4 py-2.5 tabular-nums text-ink-2">{m.scanned_qty}</td>
                      <td className="px-4 py-2.5 tabular-nums text-danger">{m.missing_qty}</td>
                      <td className="px-4 py-2.5 text-ink-3">{formatDate(m.completed_at)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </>
        )}

        {tab === 'damaged-lost' && (
          <>
            {!error && loading && report.damaged_lost.length === 0 && <p className="px-6 py-4 text-xs text-ink-4">Loading…</p>}
            {!error && !loading && report.damaged_lost.length === 0 && (
              <p className="px-6 py-4 text-xs text-ink-4">Nothing damaged or lost across any completed booking.</p>
            )}
            {report.damaged_lost.length > 0 && (
              <table className="w-full text-xs">
                <thead className="sticky top-0 z-10 bg-sunken border-b border-line">
                  <tr>
                    {['SKU', 'Product', 'Type', 'Qty', 'Reason', 'Scanned by', 'Date'].map(h => (
                      <th key={h} className="px-4 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">{h}</th>
                    ))}
                  </tr>
                </thead>
                <tbody className="divide-y divide-line/60">
                  {report.damaged_lost.map(d => (
                    <tr key={d.scan_id} className="hover:bg-hover/60 transition-colors">
                      <td className="px-4 py-2.5 font-mono text-ink-2">{d.sku_code ?? '—'}</td>
                      <td className="px-4 py-2.5 text-ink">{d.product_name ?? '—'}</td>
                      <td className="px-4 py-2.5 capitalize text-danger">{d.action_type}</td>
                      <td className="px-4 py-2.5 tabular-nums text-ink-2">{d.quantity}</td>
                      <td className="px-4 py-2.5 text-ink-3">{d.reason || '—'}</td>
                      <td className="px-4 py-2.5 text-ink-3">{d.scanned_by || '—'}</td>
                      <td className="px-4 py-2.5 text-ink-3">{formatDate(d.scanned_at)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </>
        )}
      </div>
    </div>
  )
}
