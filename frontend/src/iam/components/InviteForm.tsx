import { useState } from 'react'
import { X } from 'lucide-react'
import type { MemberRole } from '../../team/types'

export function InviteForm({ onSubmit, onCancel }: { onSubmit: (email: string, role: MemberRole) => Promise<void>; onCancel: () => void }) {
  const [email, setEmail] = useState('')
  const [role, setRole] = useState<MemberRole>('customer_user')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const handleSubmit = async () => {
    setSubmitting(true)
    setError(null)
    try {
      await onSubmit(email, role)
      onCancel()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Unknown error')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="absolute right-5 top-14 z-20 w-80 bg-raised rounded-xl border border-line shadow-lg p-4">
      <div className="flex items-center justify-between mb-3">
        <h3 className="text-sm font-semibold text-ink">Invite user</h3>
        <button onClick={onCancel} aria-label="Close" className="p-1 -m-1 rounded hover:bg-hover"><X size={14} className="text-ink-3" /></button>
      </div>
      {error && <p className="text-xs text-danger mb-2">{error}</p>}
      <label className="block text-xs text-ink-3 mb-1" htmlFor="invite-email">Email</label>
      <input
        id="invite-email"
        type="email"
        value={email}
        onChange={e => setEmail(e.target.value)}
        className="w-full mb-3 px-3 py-1.5 text-xs rounded-lg bg-sunken border border-line text-ink outline-none focus:ring-1 focus:ring-focus"
      />
      <label className="block text-xs text-ink-3 mb-1" htmlFor="invite-role">Role</label>
      <select
        id="invite-role"
        value={role}
        onChange={e => setRole(e.target.value as MemberRole)}
        className="w-full mb-4 px-3 py-1.5 text-xs rounded-lg bg-sunken border border-line text-ink outline-none focus-visible:ring-2 focus-visible:ring-focus/60"
      >
        <option value="customer_user">User</option>
        <option value="customer_owner">Owner</option>
      </select>
      <button
        onClick={handleSubmit}
        disabled={submitting || !email}
        className="w-full py-1.5 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover disabled:opacity-50"
      >
        {submitting ? 'Sending…' : 'Send invite'}
      </button>
    </div>
  )
}
