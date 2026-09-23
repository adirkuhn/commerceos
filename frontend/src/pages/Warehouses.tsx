import { useState } from 'react'
import { Plus, Search, Warehouse as WarehouseIcon, Trash2, Edit3, ArrowLeft } from 'lucide-react'
import { useAuth } from '../auth/AuthContext'
import Pagination from '../components/common/Pagination'
import { useRoutedEditor, NEW_RECORD } from '../routing/useRoutedEditor'
import { useQueryPage, useQueryString } from '../routing/useQueryState'
import { RecordLoading, RecordNotFound } from '../components/common/RecordStatus'
import { useWarehouses } from '../inventory/hooks/useWarehouses'
import type { Warehouse, WarehouseInput } from '../inventory/types'
import { useConfirm } from '../components/common/useConfirm'

const PAGE_SIZE = 20

export default function Warehouses() {
  const { confirm, confirmDialog } = useConfirm()
  const { token } = useAuth()
  const { warehouses, loading, error, create, update, remove } = useWarehouses(token ?? '')

  // Search, page and the open editor (/warehouses/new, /warehouses/:id) live in the
  // URL -- refresh and Back keep the user in place.
  const [searchParam, setSearch] = useQueryString('q', { resets: ['page'] })
  const search = searchParam ?? ''
  const [page, setPage] = useQueryPage()
  const { editingId, openNew, openEdit, close } = useRoutedEditor('/warehouses')
  const [deleteError, setDeleteError] = useState('')

  function handleSearchChange(value: string) {
    setSearch(value)
  }

  const filtered = warehouses.filter(w => w.name.toLowerCase().includes(search.toLowerCase()))
  const paged = filtered.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE)

  async function handleDelete(w: Warehouse) {
    if (!(await confirm({ title: `Delete warehouse "${w.name}"?`, message: 'This cannot be undone.', confirmLabel: 'Delete warehouse', destructive: true }))) return
    setDeleteError('')
    try {
      await remove(w.id)
    } catch (err) {
      setDeleteError(err instanceof Error ? err.message : 'Unknown error')
    }
  }

  if (editingId !== null) {
    const existing = editingId === NEW_RECORD ? null : warehouses.find(w => w.id === editingId) ?? null
    if (editingId !== NEW_RECORD && !existing) {
      return loading ? <RecordLoading /> : <RecordNotFound label="Warehouse" onBack={close} />
    }
    return (
      <WarehouseEditPage
        key={editingId}
        warehouse={existing}
        onCreate={create}
        onUpdate={update}
        onClose={close}
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
            placeholder="Search by name..."
            className="w-full pl-8 pr-3 py-1.5 text-xs rounded-lg bg-raised border-line border text-ink placeholder-ink-4 outline-none focus:ring-1 focus:ring-focus"
          />
        </div>
        <div className="flex items-center gap-2 ml-auto">
          <button
            onClick={openNew}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-accent-solid text-white text-xs hover:bg-accent-solid-hover transition-colors font-medium"
          >
            <Plus size={13} /> New Warehouse
          </button>
        </div>
      </div>

      {/* Stats strip */}
      <div className="flex items-center gap-6 px-5 py-2.5 border-b border-line bg-surface shrink-0">
        <span className="text-xs text-ink-4">{filtered.length} warehouses</span>
        {(error || deleteError) && <span className="text-xs text-danger ml-auto">{error || deleteError}</span>}
      </div>

      {/* Table */}
      <div className="flex-1 overflow-auto">
        <table className="w-full text-xs">
          <thead className="sticky top-0 z-10 bg-sunken border-b border-line">
            <tr>
              <th className="px-2 py-2.5 w-8" />
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">Warehouse</th>
              <th className="px-3 py-2.5 w-8" />
            </tr>
          </thead>
          <tbody className="divide-y divide-line/60">
            {loading && (
              <tr><td colSpan={3} className="px-4 py-6 text-center text-ink-4">Loading…</td></tr>
            )}
            {!loading && filtered.length === 0 && (
              <tr><td colSpan={3} className="px-4 py-6 text-center text-ink-4">No warehouses yet</td></tr>
            )}
            {paged.map(w => (
              <tr
                key={w.id}
                onClick={() => openEdit(w.id)}
                className="group hover:bg-hover/60 transition-colors cursor-pointer"
              >
                <td className="px-2 py-2.5">
                  <div className="w-6 h-6 rounded dark:bg-night-700 bg-gray-200 flex items-center justify-center">
                    <WarehouseIcon size={12} className="text-ink-4" />
                  </div>
                </td>
                <td className="px-3 py-2.5 max-w-[280px]">
                  <p className="font-medium text-ink truncate">
                    {w.name}
                    {!w.active && <span className="ml-2 px-1.5 py-0.5 rounded-full text-2xs font-medium dark:bg-gray-500/15 bg-gray-100 text-ink-3 align-middle">inactive</span>}
                  </p>
                  {(w.city || w.country) && (
                    <p className="text-2xs text-ink-4 truncate">{[w.city, w.country].filter(Boolean).join(', ')}</p>
                  )}
                </td>
                <td className="px-3 py-2.5">
                  <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 focus-within:opacity-100 transition-opacity">
                    <button
                      onClick={e => { e.stopPropagation(); openEdit(w.id) }}
                      className="w-6 h-6 flex items-center justify-center rounded hover:bg-hover transition-colors"
                      title="Edit"
                    >
                      <Edit3 size={12} className="text-ink-3" />
                    </button>
                    <button
                      onClick={e => { e.stopPropagation(); handleDelete(w) }}
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
      {confirmDialog}
    </div>
  )
}

// ── Warehouse Edit Page ──────────────────────────────────────────────────────

interface WarehouseEditPageProps {
  warehouse: Warehouse | null
  onCreate: (input: WarehouseInput) => Promise<Warehouse>
  onUpdate: (id: string, input: WarehouseInput) => Promise<Warehouse>
  onClose: () => void
}

function WarehouseEditPage({ warehouse, onCreate, onUpdate, onClose }: WarehouseEditPageProps) {
  const [name, setName] = useState(warehouse?.name ?? '')
  const [address1, setAddress1] = useState(warehouse?.address1 ?? '')
  const [address2, setAddress2] = useState(warehouse?.address2 ?? '')
  const [city, setCity] = useState(warehouse?.city ?? '')
  const [province, setProvince] = useState(warehouse?.province ?? '')
  const [zip, setZip] = useState(warehouse?.zip ?? '')
  const [country, setCountry] = useState(warehouse?.country ?? '')
  const [phone, setPhone] = useState(warehouse?.phone ?? '')
  const [active, setActive] = useState(warehouse?.active ?? true)
  const [saving, setSaving] = useState(false)
  const [formError, setFormError] = useState('')

  async function handleSave(e: React.FormEvent) {
    e.preventDefault()
    setFormError('')
    setSaving(true)
    try {
      const input: WarehouseInput = { name, address1, address2, city, province, zip, country, phone, active }
      if (warehouse) {
        await onUpdate(warehouse.id, input)
      } else {
        await onCreate(input)
      }
      onClose()
    } catch (err) {
      setFormError(err instanceof Error ? err.message : 'Failed to save warehouse')
    } finally {
      setSaving(false)
    }
  }

  const labelClass = 'block text-2xs font-medium text-ink-4 uppercase tracking-wider mb-1.5'
  const inputClass = 'w-full px-3 py-2 text-xs rounded-lg bg-raised border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus transition-shadow'

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-6 py-4 border-b border-line bg-sunken shrink-0">
        <div>
          <button
            onClick={onClose}
            className="flex items-center gap-1 text-2xs font-medium text-ink-4 hover:text-ink-2 transition-colors mb-1"
          >
            <ArrowLeft size={11} /> Warehouses
          </button>
          <h1 className="text-xl font-semibold text-ink leading-tight">
            {warehouse ? name || 'Edit Warehouse' : 'New Warehouse'}
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
            type="submit" form="warehouse-form" disabled={saving || !name.trim()}
            className="px-4 py-2 rounded-lg bg-accent-solid text-white text-xs font-semibold hover:bg-accent-solid-hover disabled:opacity-50 transition-colors"
          >
            {saving ? 'Saving…' : warehouse ? 'Save' : 'Create Warehouse'}
          </button>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto">
        <div className="max-w-2xl mx-auto p-6 space-y-5">
          <form id="warehouse-form" onSubmit={handleSave} className="space-y-5">
            <div className="rounded-xl border border-line p-4 space-y-3">
              <div className="flex items-center justify-between">
                <h3 className="text-xs font-semibold text-ink">General</h3>
                <label className="relative flex items-center gap-2 cursor-pointer shrink-0">
                  <span className="text-2xs font-medium text-ink-3">Active</span>
                  <span className="relative inline-flex items-center">
                    <input
                      type="checkbox"
                      role="switch"
                      aria-label="Active"
                      checked={active}
                      onChange={e => setActive(e.target.checked)}
                      className="sr-only peer"
                    />
                    <span className="w-8 h-4 dark:bg-night-700 bg-gray-300 rounded-full peer peer-checked:bg-accent-solid transition-colors" />
                    <span className="absolute left-0.5 top-0.5 w-3 h-3 bg-white rounded-full transition-all peer-checked:translate-x-4" />
                  </span>
                </label>
              </div>
              <div>
                <label htmlFor="warehouse-name" className={labelClass}>Name</label>
                <input
                  id="warehouse-name"
                  type="text" required autoFocus value={name} onChange={e => setName(e.target.value)}
                  className={inputClass}
                />
              </div>
              {formError && (
                <p className="text-xs text-danger bg-red-50 dark:bg-red-500/10 rounded-lg px-2 py-1.5">{formError}</p>
              )}
            </div>

            <div className="rounded-xl border border-line p-4 space-y-3">
              <div>
                <h3 className="text-xs font-semibold text-ink">Address</h3>
                <p className="text-2xs text-ink-4 mt-0.5">Where imported and stocked inventory physically lives.</p>
              </div>
              <div>
                <label htmlFor="warehouse-address1" className={labelClass}>Address line 1</label>
                <input
                  id="warehouse-address1"
                  type="text" value={address1} onChange={e => setAddress1(e.target.value)}
                  className={inputClass}
                />
              </div>
              <div>
                <label htmlFor="warehouse-address2" className={labelClass}>Address line 2</label>
                <input
                  id="warehouse-address2"
                  type="text" value={address2} onChange={e => setAddress2(e.target.value)}
                  className={inputClass}
                />
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label htmlFor="warehouse-city" className={labelClass}>City</label>
                  <input
                    id="warehouse-city"
                    type="text" value={city} onChange={e => setCity(e.target.value)}
                    className={inputClass}
                  />
                </div>
                <div>
                  <label htmlFor="warehouse-province" className={labelClass}>Province / State</label>
                  <input
                    id="warehouse-province"
                    type="text" value={province} onChange={e => setProvince(e.target.value)}
                    className={inputClass}
                  />
                </div>
                <div>
                  <label htmlFor="warehouse-zip" className={labelClass}>ZIP / Postal code</label>
                  <input
                    id="warehouse-zip"
                    type="text" value={zip} onChange={e => setZip(e.target.value)}
                    className={inputClass}
                  />
                </div>
                <div>
                  <label htmlFor="warehouse-country" className={labelClass}>Country</label>
                  <input
                    id="warehouse-country"
                    type="text" value={country} onChange={e => setCountry(e.target.value)}
                    className={inputClass}
                  />
                </div>
              </div>
              <div>
                <label htmlFor="warehouse-phone" className={labelClass}>Phone</label>
                <input
                  id="warehouse-phone"
                  type="tel" value={phone} onChange={e => setPhone(e.target.value)}
                  className={inputClass}
                />
              </div>
            </div>
          </form>
        </div>
      </div>
    </div>
  )
}
