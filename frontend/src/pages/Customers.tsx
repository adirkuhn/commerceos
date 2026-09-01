import { useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { Plus, Search, User, Trash2, Edit3, ArrowLeft } from 'lucide-react'
import clsx from 'clsx'
import { useAuth } from '../auth/AuthContext'
import Pagination from '../components/common/Pagination'
import { useCustomers } from '../customers/hooks/useCustomers'
import type { Customer, CustomerInput, Address } from '../customers/types'

const PAGE_SIZE = 20

function fullName(c: Customer): string {
  return [c.first_name, c.last_name].filter(Boolean).join(' ') || c.email
}

export default function Customers() {
  const { token } = useAuth()
  const { customers, loading, error, create, update, remove } = useCustomers(token ?? '')

  const [search, setSearch] = useState('')
  const [page, setPage] = useState(1)
  // The URL is the source of truth for "which customer is open" (?id=<uuid>,
  // or ?id=new) -- survives a refresh, and is how global search (Cmd+K)
  // deep-links straight to a specific customer. Same convention as
  // Products.tsx's editingId.
  const [searchParams, setSearchParams] = useSearchParams()
  const editingId = searchParams.get('id')
  const [deleteError, setDeleteError] = useState('')

  function handleSearchChange(value: string) {
    setSearch(value)
    setPage(1)
  }

  const filtered = customers.filter(c =>
    fullName(c).toLowerCase().includes(search.toLowerCase()) || c.email.toLowerCase().includes(search.toLowerCase()),
  )
  const paged = filtered.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE)

  async function handleDelete(c: Customer) {
    if (!window.confirm(`Delete customer "${fullName(c)}"? This cannot be undone.`)) return
    setDeleteError('')
    try {
      await remove(c.id)
    } catch (err) {
      setDeleteError(err instanceof Error ? err.message : 'Unknown error')
    }
  }

  if (editingId !== null) {
    const existing = editingId === 'new' ? null : customers.find(c => c.id === editingId) ?? null
    return (
      <CustomerEditPage
        customer={existing}
        onCreate={create}
        onUpdate={update}
        onClose={() => setSearchParams({})}
      />
    )
  }

  return (
    <div className="flex flex-col h-full">
      {/* Toolbar */}
      <div className="flex items-center gap-3 px-5 py-3 border-b dark:border-[#2D3748] border-gray-200 dark:bg-[#111827] bg-gray-50 shrink-0">
        <div className="relative flex-1 max-w-sm">
          <Search size={13} className="absolute left-3 top-1/2 -translate-y-1/2 dark:text-gray-500 text-gray-400" />
          <input
            value={search}
            onChange={e => handleSearchChange(e.target.value)}
            placeholder="Search by name or email..."
            className="w-full pl-8 pr-3 py-1.5 text-xs rounded-lg dark:bg-[#1F2937] bg-white dark:border-[#2D3748] border-gray-200 border dark:text-gray-200 text-gray-800 dark:placeholder-gray-500 placeholder-gray-400 outline-none focus:ring-1 focus:ring-blue-500"
          />
        </div>
        <div className="flex items-center gap-2 ml-auto">
          <button
            onClick={() => setSearchParams({ id: 'new' })}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-blue-600 text-white text-xs hover:bg-blue-700 transition-colors font-medium"
          >
            <Plus size={13} /> Add Customer
          </button>
        </div>
      </div>

      {/* Stats strip */}
      <div className="flex items-center gap-6 px-5 py-2.5 border-b dark:border-[#2D3748] border-gray-200 dark:bg-[#111827] bg-white shrink-0">
        <span className="text-xs dark:text-gray-500 text-gray-400">{filtered.length} customer{filtered.length === 1 ? '' : 's'}</span>
        {(error || deleteError) && <span className="text-xs text-red-400 ml-auto">{error || deleteError}</span>}
      </div>

      {/* Table */}
      <div className="flex-1 overflow-auto">
        <table className="w-full text-xs">
          <thead className="sticky top-0 z-10 dark:bg-[#111827] bg-gray-50 border-b dark:border-[#2D3748] border-gray-200">
            <tr>
              <th className="px-2 py-2.5 w-8" />
              <th className="px-3 py-2.5 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider">Customer</th>
              <th className="px-3 py-2.5 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider">Location</th>
              <th className="px-3 py-2.5 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider">Status</th>
              <th className="px-3 py-2.5 w-16" />
            </tr>
          </thead>
          <tbody className="divide-y dark:divide-[#2D3748]/60 divide-gray-100">
            {loading && (
              <tr><td colSpan={5} className="px-4 py-6 text-center dark:text-gray-500 text-gray-400">Loading…</td></tr>
            )}
            {!loading && filtered.length === 0 && (
              <tr><td colSpan={5} className="px-4 py-6 text-center dark:text-gray-500 text-gray-400">No customers yet</td></tr>
            )}
            {paged.map(c => (
              <tr
                key={c.id}
                onClick={() => setSearchParams({ id: c.id })}
                className="group hover:dark:bg-[#1F2937]/60 hover:bg-gray-50 transition-colors cursor-pointer"
              >
                <td className="px-2 py-2.5">
                  <div className="w-6 h-6 rounded dark:bg-[#2D3748] bg-gray-200 flex items-center justify-center">
                    <User size={12} className="dark:text-gray-500 text-gray-400" />
                  </div>
                </td>
                <td className="px-3 py-2.5 max-w-[280px]">
                  <p className="font-medium dark:text-gray-100 text-gray-900 truncate">{fullName(c)}</p>
                  <p className="text-[10px] dark:text-gray-500 text-gray-400 truncate">{c.email}</p>
                </td>
                <td className="px-3 py-2.5 dark:text-gray-400 text-gray-500">
                  {[c.billing?.city, c.billing?.country].filter(Boolean).join(', ') || '—'}
                </td>
                <td className="px-3 py-2.5">
                  <span className={clsx(
                    'px-2 py-0.5 rounded-full text-[10px] font-medium',
                    c.is_paying_customer ? 'bg-green-500/15 text-green-400' : 'bg-gray-500/15 text-gray-400',
                  )}>
                    {c.is_paying_customer ? 'Paying' : 'No orders yet'}
                  </span>
                </td>
                <td className="px-3 py-2.5">
                  <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                    <button
                      onClick={e => { e.stopPropagation(); setSearchParams({ id: c.id }) }}
                      className="w-6 h-6 flex items-center justify-center rounded dark:hover:bg-[#2D3748] hover:bg-gray-200 transition-colors"
                      title="Edit"
                    >
                      <Edit3 size={12} className="dark:text-gray-400 text-gray-500" />
                    </button>
                    <button
                      onClick={e => { e.stopPropagation(); handleDelete(c) }}
                      className="w-6 h-6 flex items-center justify-center rounded dark:hover:bg-[#2D3748] hover:bg-gray-200 transition-colors"
                      title="Delete"
                    >
                      <Trash2 size={12} className="dark:text-gray-400 text-gray-500 hover:text-red-400" />
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

// ── Customer Edit Page ───────────────────────────────────────────────────────

interface CustomerEditPageProps {
  customer: Customer | null
  onCreate: (input: CustomerInput) => Promise<Customer>
  onUpdate: (id: string, input: CustomerInput) => Promise<Customer>
  onClose: () => void
}

function emptyAddress(): Address {
  return { first_name: '', last_name: '', company: '', address_1: '', address_2: '', city: '', state: '', postcode: '', country: '', email: '', phone: '' }
}

function CustomerEditPage({ customer, onCreate, onUpdate, onClose }: CustomerEditPageProps) {
  const [firstName, setFirstName] = useState(customer?.first_name ?? '')
  const [lastName, setLastName] = useState(customer?.last_name ?? '')
  const [email, setEmail] = useState(customer?.email ?? '')
  const [billing, setBilling] = useState<Address>({ ...emptyAddress(), ...customer?.billing })
  const [shipping, setShipping] = useState<Address>({ ...emptyAddress(), ...customer?.shipping })
  const [isPayingCustomer, setIsPayingCustomer] = useState(customer?.is_paying_customer ?? false)
  const [saving, setSaving] = useState(false)
  const [formError, setFormError] = useState('')

  async function handleSave(e: React.FormEvent) {
    e.preventDefault()
    setFormError('')
    setSaving(true)
    try {
      const input: CustomerInput = {
        email, first_name: firstName, last_name: lastName, billing, shipping, is_paying_customer: isPayingCustomer,
      }
      if (customer) {
        await onUpdate(customer.id, input)
      } else {
        await onCreate(input)
      }
      onClose()
    } catch (err) {
      setFormError(err instanceof Error ? err.message : 'Failed to save customer')
    } finally {
      setSaving(false)
    }
  }

  const labelClass = 'block text-[10px] font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider mb-1.5'
  const inputClass = 'w-full px-3 py-2 text-xs rounded-lg dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none focus:ring-1 focus:ring-blue-500 transition-shadow'

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-6 py-4 border-b dark:border-[#2D3748] border-gray-200 dark:bg-[#111827] bg-gray-50 shrink-0">
        <div>
          <button
            onClick={onClose}
            className="flex items-center gap-1 text-[11px] font-medium dark:text-gray-500 text-gray-400 hover:dark:text-gray-300 hover:text-gray-600 transition-colors mb-1"
          >
            <ArrowLeft size={11} /> Customer
          </button>
          <h1 className="text-xl font-semibold dark:text-gray-100 text-gray-900 leading-tight">
            {customer ? [firstName, lastName].filter(Boolean).join(' ') || 'Edit Customer' : 'New Customer'}
          </h1>
        </div>
        <div className="flex items-center gap-2">
          <button
            type="button" onClick={onClose}
            className="px-4 py-2 rounded-lg border dark:border-[#2D3748] border-gray-200 dark:bg-[#1F2937] bg-white text-xs font-semibold dark:text-gray-300 text-gray-700 hover:dark:bg-[#2D3748] hover:bg-gray-50 transition-colors"
          >
            Discard
          </button>
          <button
            type="submit" form="customer-form" disabled={saving || !email.trim()}
            className="px-4 py-2 rounded-lg bg-blue-600 text-white text-xs font-semibold hover:bg-blue-700 disabled:opacity-50 transition-colors"
          >
            {saving ? 'Saving…' : customer ? 'Save' : 'Create Customer'}
          </button>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto">
        <div className="max-w-2xl mx-auto p-6 space-y-5">
          <form id="customer-form" onSubmit={handleSave} className="space-y-5">
            <div className="rounded-xl border dark:border-[#2D3748] border-gray-200 p-4 space-y-3">
              <div className="flex items-center justify-between">
                <h3 className="text-xs font-semibold dark:text-gray-100 text-gray-900">General</h3>
                <label className="flex items-center gap-1.5 text-[10px] dark:text-gray-400 text-gray-500">
                  <input type="checkbox" checked={isPayingCustomer} onChange={e => setIsPayingCustomer(e.target.checked)} />
                  Paying customer
                </label>
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label htmlFor="customer-first-name" className={labelClass}>First name</label>
                  <input id="customer-first-name" type="text" autoFocus value={firstName} onChange={e => setFirstName(e.target.value)} className={inputClass} />
                </div>
                <div>
                  <label htmlFor="customer-last-name" className={labelClass}>Last name</label>
                  <input id="customer-last-name" type="text" value={lastName} onChange={e => setLastName(e.target.value)} className={inputClass} />
                </div>
              </div>
              <div>
                <label htmlFor="customer-email" className={labelClass}>Email</label>
                <input id="customer-email" type="email" required value={email} onChange={e => setEmail(e.target.value)} className={inputClass} />
              </div>
              {formError && (
                <p className="text-xs text-red-400 bg-red-50 dark:bg-red-500/10 rounded-lg px-2 py-1.5">{formError}</p>
              )}
            </div>

            <AddressFields title="Billing address" prefix="billing" value={billing} onChange={setBilling} showEmailPhone />
            <AddressFields title="Shipping address" prefix="shipping" value={shipping} onChange={setShipping} />
          </form>
        </div>
      </div>
    </div>
  )
}

function AddressFields({ title, prefix, value, onChange, showEmailPhone }: { title: string; prefix: string; value: Address; onChange: (a: Address) => void; showEmailPhone?: boolean }) {
  const labelClass = 'block text-[10px] font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider mb-1.5'
  const inputClass = 'w-full px-3 py-2 text-xs rounded-lg dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none focus:ring-1 focus:ring-blue-500 transition-shadow'
  const set = (field: keyof Address) => (e: React.ChangeEvent<HTMLInputElement>) => onChange({ ...value, [field]: e.target.value })
  const id = (field: string) => `${prefix}-${field}`
  const label = (text: string) => `${title.replace(' address', '')} ${text}`

  return (
    <div className="rounded-xl border dark:border-[#2D3748] border-gray-200 p-4 space-y-3">
      <h3 className="text-xs font-semibold dark:text-gray-100 text-gray-900">{title}</h3>
      <div className="grid grid-cols-2 gap-3">
        <div>
          <label htmlFor={id('address-1')} className={labelClass}>{label('address line 1')}</label>
          <input id={id('address-1')} type="text" value={value.address_1 ?? ''} onChange={set('address_1')} className={inputClass} />
        </div>
        <div>
          <label htmlFor={id('address-2')} className={labelClass}>{label('address line 2')}</label>
          <input id={id('address-2')} type="text" value={value.address_2 ?? ''} onChange={set('address_2')} className={inputClass} />
        </div>
        <div>
          <label htmlFor={id('city')} className={labelClass}>{label('city')}</label>
          <input id={id('city')} type="text" value={value.city ?? ''} onChange={set('city')} className={inputClass} />
        </div>
        <div>
          <label htmlFor={id('state')} className={labelClass}>{label('state')}</label>
          <input id={id('state')} type="text" value={value.state ?? ''} onChange={set('state')} className={inputClass} />
        </div>
        <div>
          <label htmlFor={id('postcode')} className={labelClass}>{label('postcode')}</label>
          <input id={id('postcode')} type="text" value={value.postcode ?? ''} onChange={set('postcode')} className={inputClass} />
        </div>
        <div>
          <label htmlFor={id('country')} className={labelClass}>{label('country')}</label>
          <input id={id('country')} type="text" value={value.country ?? ''} onChange={set('country')} className={inputClass} />
        </div>
        {showEmailPhone && (
          <div>
            <label htmlFor={id('phone')} className={labelClass}>{label('phone')}</label>
            <input id={id('phone')} type="tel" value={value.phone ?? ''} onChange={set('phone')} className={inputClass} />
          </div>
        )}
      </div>
    </div>
  )
}
