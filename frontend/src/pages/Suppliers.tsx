import { useState } from 'react'
import { Plus, Search, Truck, Trash2, Edit3, ArrowLeft } from 'lucide-react'
import clsx from 'clsx'
import { useAuth } from '../auth/AuthContext'
import Pagination from '../components/common/Pagination'
import { useSuppliers } from '../suppliers/hooks/useSuppliers'
import type { Supplier, SupplierInput, SupplierStatus } from '../suppliers/types'
import { useQueryPage, useQueryString } from '../routing/useQueryState'
import { useRoutedEditor, NEW_RECORD } from '../routing/useRoutedEditor'
import { RecordLoading, RecordNotFound } from '../components/common/RecordStatus'

const PAGE_SIZE = 20

const STATUS_STYLE: Record<SupplierStatus, string> = {
  active: 'bg-green-500/15 text-success',
  inactive: 'bg-gray-500/15 text-ink-4',
}

export default function Suppliers() {
  const { token } = useAuth()
  const { suppliers, loading, error, create, update, remove } = useSuppliers(token ?? '')

  // Search and page live in the URL (?q=, ?page=) -- refresh and Back keep them.
  const [searchParam, setSearch] = useQueryString('q', { resets: ['page'] })
  const search = searchParam ?? ''
  const [page, setPage] = useQueryPage()
  // The URL is the source of truth for "which supplier is open"
  // (/suppliers/<uuid>, or /suppliers/new) -- survives a refresh, and is how
  // global search (Cmd+K) deep-links straight to a specific supplier. Same
  // convention as Products.tsx's editingId (see its comment for why this
  // reads location.pathname on every render instead of useParams).
  const { editingId, openEdit: openEditor, openNew, close: closeEditor } = useRoutedEditor('/suppliers')
  const [deleteError, setDeleteError] = useState('')

  function handleSearchChange(value: string) {
    setSearch(value)
  }

  const filtered = suppliers.filter(s =>
    s.name.toLowerCase().includes(search.toLowerCase()) || s.country.toLowerCase().includes(search.toLowerCase()),
  )
  const paged = filtered.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE)

  async function handleDelete(s: Supplier) {
    if (!window.confirm(`Delete supplier "${s.name}"? This cannot be undone.`)) return
    setDeleteError('')
    try {
      await remove(s.id)
    } catch (err) {
      setDeleteError(err instanceof Error ? err.message : 'Unknown error')
    }
  }

  if (editingId !== null) {
    const existing = editingId === NEW_RECORD ? null : suppliers.find(s => s.id === editingId) ?? null
    if (editingId !== NEW_RECORD && !existing) {
      return loading ? <RecordLoading /> : <RecordNotFound label="Supplier" onBack={closeEditor} />
    }
    return (
      <SupplierEditPage
        supplier={existing}
        onCreate={create}
        onUpdate={update}
        onClose={closeEditor}
      />
    )
  }

  return (
    <div className="flex flex-col h-full">
      {/* Toolbar */}
      <div className="flex items-center gap-3 px-5 py-3 border-b border-line bg-sunken shrink-0">
        <div className="relative flex-1 max-w-sm">
          <Search size={13} className="absolute left-3 top-1/2 -translate-y-1/2 text-ink-4" />
          <input
            value={search}
            onChange={e => handleSearchChange(e.target.value)}
            placeholder="Search by name or country..."
            className="w-full pl-8 pr-3 py-1.5 text-xs rounded-lg bg-raised border-line border text-ink placeholder-ink-4 outline-none focus:ring-1 focus:ring-blue-500"
          />
        </div>
        <div className="flex items-center gap-2 ml-auto">
          <button
            onClick={openNew}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-blue-600 text-white text-xs hover:bg-blue-700 transition-colors font-medium"
          >
            <Plus size={13} /> Add Supplier
          </button>
        </div>
      </div>

      {/* Stats strip */}
      <div className="flex items-center gap-6 px-5 py-2.5 border-b border-line bg-surface shrink-0">
        <span className="text-xs text-ink-4">{filtered.length} supplier{filtered.length === 1 ? '' : 's'}</span>
        {(error || deleteError) && <span className="text-xs text-danger ml-auto">{error || deleteError}</span>}
      </div>

      {/* Table */}
      <div className="flex-1 overflow-auto">
        <table className="w-full text-xs">
          <thead className="sticky top-0 z-10 bg-sunken border-b border-line">
            <tr>
              <th className="px-2 py-2.5 w-8" />
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">Supplier</th>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">Country</th>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">Lead Time</th>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">Status</th>
              <th className="px-3 py-2.5 w-16" />
            </tr>
          </thead>
          <tbody className="divide-y divide-line/60">
            {loading && (
              <tr><td colSpan={6} className="px-4 py-6 text-center text-ink-4">Loading…</td></tr>
            )}
            {!loading && filtered.length === 0 && (
              <tr><td colSpan={6} className="px-4 py-6 text-center text-ink-4">No suppliers yet</td></tr>
            )}
            {paged.map(s => (
              <tr
                key={s.id}
                onClick={() => openEditor(s.id)}
                className="group hover:bg-hover/60 transition-colors cursor-pointer"
              >
                <td className="px-2 py-2.5">
                  <div className="w-6 h-6 rounded dark:bg-night-700 bg-gray-200 flex items-center justify-center">
                    <Truck size={12} className="text-ink-4" />
                  </div>
                </td>
                <td className="px-3 py-2.5 max-w-[280px]">
                  <p className="font-medium text-ink truncate">{s.name}</p>
                  {s.email && <p className="text-2xs text-ink-4 truncate">{s.email}</p>}
                </td>
                <td className="px-3 py-2.5 text-ink-3">{s.country || '—'}</td>
                <td className="px-3 py-2.5 text-ink-3">{s.lead_time_days != null ? `${s.lead_time_days}d` : '—'}</td>
                <td className="px-3 py-2.5">
                  <span className={clsx('px-2 py-0.5 rounded-full text-2xs font-medium capitalize', STATUS_STYLE[s.status])}>
                    {s.status}
                  </span>
                </td>
                <td className="px-3 py-2.5">
                  <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 focus-within:opacity-100 transition-opacity">
                    <button
                      onClick={e => { e.stopPropagation(); openEditor(s.id) }}
                      className="w-6 h-6 flex items-center justify-center rounded hover:bg-hover transition-colors"
                      title="Edit"
                    >
                      <Edit3 size={12} className="text-ink-3" />
                    </button>
                    <button
                      onClick={e => { e.stopPropagation(); handleDelete(s) }}
                      className="w-6 h-6 flex items-center justify-center rounded hover:bg-hover transition-colors"
                      title="Delete"
                    >
                      <Trash2 size={12} className="text-ink-3 hover:text-danger" />
                    </button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <Pagination page={page} pageSize={PAGE_SIZE} totalItems={filtered.length} onPageChange={setPage} />
    </div>
  )
}

// ── Supplier Edit Page ────────────────────────────────────────────────────────

interface SupplierEditPageProps {
  supplier: Supplier | null
  onCreate: (input: SupplierInput) => Promise<Supplier>
  onUpdate: (id: string, input: SupplierInput) => Promise<Supplier>
  onClose: () => void
}

function SupplierEditPage({ supplier, onCreate, onUpdate, onClose }: SupplierEditPageProps) {
  const [name, setName] = useState(supplier?.name ?? '')
  const [email, setEmail] = useState(supplier?.email ?? '')
  const [phone, setPhone] = useState(supplier?.phone ?? '')
  const [country, setCountry] = useState(supplier?.country ?? '')
  const [leadTimeDays, setLeadTimeDays] = useState(supplier?.lead_time_days != null ? String(supplier.lead_time_days) : '')
  const [status, setStatus] = useState<SupplierStatus>(supplier?.status ?? 'active')
  const [notes, setNotes] = useState(supplier?.notes ?? '')
  const [saving, setSaving] = useState(false)
  const [formError, setFormError] = useState('')

  // A deep link (Cmd+K search, direct URL, hard refresh) mounts this page
  // before useSuppliers' list fetch resolves, so the useState initializers
  // above run with supplier still null. Once the list arrives and `supplier`
  // is passed in for the first time, re-seed the form from it -- without
  // this, the fields stay blank forever since useState never re-runs.
  const [hydrated, setHydrated] = useState(false)
  if (!hydrated && supplier) {
    setName(supplier.name ?? '')
    setEmail(supplier.email ?? '')
    setPhone(supplier.phone ?? '')
    setCountry(supplier.country ?? '')
    setLeadTimeDays(supplier.lead_time_days != null ? String(supplier.lead_time_days) : '')
    setStatus(supplier.status ?? 'active')
    setNotes(supplier.notes ?? '')
    setHydrated(true)
  }

  async function handleSave(e: React.FormEvent) {
    e.preventDefault()
    setFormError('')
    setSaving(true)
    try {
      const input: SupplierInput = {
        name, email, phone, country,
        lead_time_days: leadTimeDays.trim() === '' ? null : parseInt(leadTimeDays, 10),
        status, notes,
      }
      if (supplier) {
        await onUpdate(supplier.id, input)
      } else {
        await onCreate(input)
      }
      onClose()
    } catch (err) {
      setFormError(err instanceof Error ? err.message : 'Failed to save supplier')
    } finally {
      setSaving(false)
    }
  }

  const labelClass = 'block text-2xs font-medium text-ink-4 uppercase tracking-wider mb-1.5'
  const inputClass = 'w-full px-3 py-2 text-xs rounded-lg bg-raised border border-line text-ink-2 outline-none focus:ring-1 focus:ring-blue-500 transition-shadow'

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-6 py-4 border-b border-line bg-sunken shrink-0">
        <div>
          <button
            onClick={onClose}
            className="flex items-center gap-1 text-2xs font-medium text-ink-4 hover:text-ink-2 transition-colors mb-1"
          >
            <ArrowLeft size={11} /> Suppliers
          </button>
          <h1 className="text-xl font-semibold text-ink leading-tight">
            {supplier ? name || 'Edit Supplier' : 'New Supplier'}
          </h1>
        </div>
        <div className="flex items-center gap-2">
          <button
            type="button" onClick={onClose}
            className="px-4 py-2 rounded-lg border border-line bg-raised text-xs font-semibold text-ink-2 hover:bg-hover transition-colors"
          >
            Discard
          </button>
          <button
            type="submit" form="supplier-form" disabled={saving || !name.trim()}
            className="px-4 py-2 rounded-lg bg-blue-600 text-white text-xs font-semibold hover:bg-blue-700 disabled:opacity-50 transition-colors"
          >
            {saving ? 'Saving…' : supplier ? 'Save' : 'Create Supplier'}
          </button>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto">
        <div className="max-w-2xl mx-auto p-6 space-y-5">
          <form id="supplier-form" onSubmit={handleSave} className="space-y-5">
            <div className="rounded-xl border border-line p-4 space-y-3">
              <div className="flex items-center justify-between">
                <h3 className="text-xs font-semibold text-ink">General</h3>
                <select
                  aria-label="Status"
                  value={status}
                  onChange={e => setStatus(e.target.value as SupplierStatus)}
                  className="px-2 py-1 text-2xs rounded-lg bg-raised border border-line text-ink-2 outline-none focus-visible:ring-2 focus-visible:ring-focus/60"
                >
                  <option value="active">Active</option>
                  <option value="inactive">Inactive</option>
                </select>
              </div>
              <div>
                <label htmlFor="supplier-name" className={labelClass}>Name</label>
                <input
                  id="supplier-name"
                  type="text" required autoFocus value={name} onChange={e => setName(e.target.value)}
                  className={inputClass}
                />
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label htmlFor="supplier-email" className={labelClass}>Email</label>
                  <input
                    id="supplier-email"
                    type="email" value={email} onChange={e => setEmail(e.target.value)}
                    className={inputClass}
                  />
                </div>
                <div>
                  <label htmlFor="supplier-phone" className={labelClass}>Phone</label>
                  <input
                    id="supplier-phone"
                    type="tel" value={phone} onChange={e => setPhone(e.target.value)}
                    className={inputClass}
                  />
                </div>
                <div>
                  <label htmlFor="supplier-country" className={labelClass}>Country</label>
                  <input
                    id="supplier-country"
                    type="text" value={country} onChange={e => setCountry(e.target.value)}
                    className={inputClass}
                  />
                </div>
                <div>
                  <label htmlFor="supplier-lead-time" className={labelClass}>Lead time (days)</label>
                  <input
                    id="supplier-lead-time"
                    type="number" min="0" value={leadTimeDays} onChange={e => setLeadTimeDays(e.target.value)}
                    className={inputClass}
                  />
                </div>
              </div>
              <div>
                <label htmlFor="supplier-notes" className={labelClass}>Notes</label>
                <textarea
                  id="supplier-notes"
                  rows={3} value={notes} onChange={e => setNotes(e.target.value)}
                  className={inputClass}
                />
              </div>
              {formError && (
                <p className="text-xs text-danger bg-red-50 dark:bg-red-500/10 rounded-lg px-2 py-1.5">{formError}</p>
              )}
            </div>
          </form>
        </div>
      </div>
    </div>
  )
}
