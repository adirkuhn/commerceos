import { useState } from 'react'
import { Plus, Search, Shield, X, CheckCircle, XCircle } from 'lucide-react'
import clsx from 'clsx'
import { useAuth } from '../auth/AuthContext'
import { useCompanyMembers } from '../team/hooks/useCompanyMembers'
import { useCompanyInvites } from '../team/hooks/useCompanyInvites'
import { useGroups } from '../iam/hooks/useGroups'
import { useGroupMembers } from '../iam/hooks/useGroupMembers'
import { useGroupPermissions } from '../iam/hooks/useGroupPermissions'
import { GroupList } from '../iam/components/GroupList'
import { GroupForm } from '../iam/components/GroupForm'
import type { MemberRole, CompanyMember } from '../team/types'
import type { Group } from '../iam/types'

const ROLE_LABELS: Record<MemberRole, string> = { customer_owner: 'Owner', customer_user: 'User' }
const ROLE_COLORS: Record<MemberRole, string> = {
  customer_owner: 'bg-red-500/15 text-red-400',
  customer_user: 'bg-gray-500/15 text-gray-400',
}

function fullName(m: CompanyMember): string {
  return [m.first_name, m.last_name].filter(Boolean).join(' ') || m.email
}

function InviteForm({ onSubmit, onCancel }: { onSubmit: (email: string, role: MemberRole) => Promise<void>; onCancel: () => void }) {
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
    <div className="absolute right-5 top-14 z-20 w-80 dark:bg-[#1F2937] bg-white rounded-xl border dark:border-[#2D3748] border-gray-200 shadow-lg p-4">
      <div className="flex items-center justify-between mb-3">
        <h3 className="text-sm font-semibold dark:text-gray-100 text-gray-900">Invite User</h3>
        <button onClick={onCancel}><X size={14} className="dark:text-gray-400 text-gray-500" /></button>
      </div>
      {error && <p className="text-xs text-red-400 mb-2">{error}</p>}
      <label className="block text-xs dark:text-gray-400 text-gray-500 mb-1" htmlFor="invite-email">Email</label>
      <input
        id="invite-email"
        type="email"
        value={email}
        onChange={e => setEmail(e.target.value)}
        className="w-full mb-3 px-3 py-1.5 text-xs rounded-lg dark:bg-[#111827] bg-gray-50 border dark:border-[#2D3748] border-gray-200 dark:text-gray-200 text-gray-800 outline-none focus:ring-1 focus:ring-blue-500"
      />
      <label className="block text-xs dark:text-gray-400 text-gray-500 mb-1" htmlFor="invite-role">Role</label>
      <select
        id="invite-role"
        value={role}
        onChange={e => setRole(e.target.value as MemberRole)}
        className="w-full mb-4 px-3 py-1.5 text-xs rounded-lg dark:bg-[#111827] bg-gray-50 border dark:border-[#2D3748] border-gray-200 dark:text-gray-200 text-gray-800 outline-none"
      >
        <option value="customer_user">User</option>
        <option value="customer_owner">Owner</option>
      </select>
      <button
        onClick={handleSubmit}
        disabled={submitting || !email}
        className="w-full py-1.5 rounded-lg bg-blue-600 text-white text-xs font-medium hover:bg-blue-700 disabled:opacity-50"
      >
        {submitting ? 'Sending…' : 'Send Invite'}
      </button>
    </div>
  )
}

function GroupDetail({ token, group, members }: { token: string; group: Group; members: CompanyMember[] }) {
  const groupMembers = useGroupMembers(token, group.id)
  const groupPerms = useGroupPermissions(token, group.id)
  const [pickUserId, setPickUserId] = useState('')
  const [newRule, setNewRule] = useState('')

  const memberByID = new Map(members.map(m => [m.user_id, m]))
  const notInGroup = members.filter(m => !groupMembers.members.includes(m.user_id))

  return (
    <div className="p-4 space-y-5">
      <div>
        <h4 className="text-xs font-semibold dark:text-gray-300 text-gray-600 uppercase tracking-wider mb-2">Members</h4>
        {groupMembers.error && <p className="text-xs text-red-400 mb-2">{groupMembers.error}</p>}
        <ul className="space-y-1 mb-3">
          {groupMembers.members.map(userId => (
            <li key={userId} className="flex items-center justify-between text-xs dark:text-gray-300 text-gray-700">
              <span>{memberByID.get(userId)?.email ?? userId}</span>
              <button onClick={() => groupMembers.remove(userId)} className="text-red-400 hover:text-red-300">Remove</button>
            </li>
          ))}
          {groupMembers.members.length === 0 && <li className="text-xs dark:text-gray-500 text-gray-400">No members yet.</li>}
        </ul>
        <div className="flex gap-2">
          <select
            aria-label="Add member to group"
            value={pickUserId}
            onChange={e => setPickUserId(e.target.value)}
            className="flex-1 px-2 py-1 text-xs rounded border dark:border-[#2D3748] border-gray-200 dark:bg-[#111827] bg-gray-50 dark:text-gray-200 text-gray-800"
          >
            <option value="">Select a member…</option>
            {notInGroup.map(m => (
              <option key={m.user_id} value={m.user_id}>{fullName(m)}</option>
            ))}
          </select>
          <button
            onClick={() => { if (pickUserId) { groupMembers.add(pickUserId); setPickUserId('') } }}
            disabled={!pickUserId}
            className="px-3 py-1 text-xs rounded bg-blue-600 text-white disabled:opacity-50"
          >
            Add
          </button>
        </div>
      </div>

      <div>
        <h4 className="text-xs font-semibold dark:text-gray-300 text-gray-600 uppercase tracking-wider mb-2">Permissions</h4>
        {groupPerms.error && <p className="text-xs text-red-400 mb-2">{groupPerms.error}</p>}
        <ul className="space-y-1 mb-3">
          {groupPerms.permissions.map(rule => (
            <li key={rule} className="flex items-center justify-between text-xs font-mono dark:text-gray-300 text-gray-700">
              <span>{rule}</span>
              <button onClick={() => groupPerms.revoke(rule)} className="text-red-400 hover:text-red-300 font-sans">Revoke</button>
            </li>
          ))}
          {groupPerms.permissions.length === 0 && <li className="text-xs dark:text-gray-500 text-gray-400">No permissions yet.</li>}
        </ul>
        <div className="flex gap-2">
          <input
            aria-label="New permission rule"
            placeholder="pms.product.create"
            value={newRule}
            onChange={e => setNewRule(e.target.value)}
            className="flex-1 px-2 py-1 text-xs rounded border dark:border-[#2D3748] border-gray-200 dark:bg-[#111827] bg-gray-50 dark:text-gray-200 text-gray-800 font-mono"
          />
          <button
            onClick={() => { if (newRule) { groupPerms.assign(newRule); setNewRule('') } }}
            disabled={!newRule}
            className="px-3 py-1 text-xs rounded bg-blue-600 text-white disabled:opacity-50"
          >
            Assign
          </button>
        </div>
      </div>
    </div>
  )
}

function GroupsPanel({ token, members }: { token: string; members: CompanyMember[] }) {
  const groups = useGroups(token)
  const [selected, setSelected] = useState<Group | null>(null)
  const [showForm, setShowForm] = useState(false)

  return (
    <div className="w-[480px] border-l dark:border-[#2D3748] border-gray-200 flex flex-col overflow-hidden">
      <div className="px-4 py-3 border-b dark:border-[#2D3748] border-gray-200 shrink-0">
        <h3 className="text-sm font-semibold dark:text-gray-100 text-gray-900">Groups & Permissions</h3>
        <p className="text-[10px] dark:text-gray-500 text-gray-400 mt-0.5">Fine-grained access on top of owner/user roles</p>
      </div>
      <div className="flex-1 overflow-auto p-4">
        {showForm ? (
          <GroupForm
            group={selected}
            onCancel={() => setShowForm(false)}
            onSubmit={async payload => {
              if (selected) await groups.update(selected.id, payload)
              else await groups.create(payload)
              setShowForm(false)
            }}
          />
        ) : (
          <GroupList
            groups={groups.groups}
            loading={groups.loading}
            error={groups.error}
            onSelect={g => setSelected(g)}
            onDelete={id => { groups.remove(id); if (selected?.id === id) setSelected(null) }}
            onCreateClick={() => { setSelected(null); setShowForm(true) }}
          />
        )}
        {selected && !showForm && (
          <div className="mt-4 border-t dark:border-[#2D3748] border-gray-200">
            <div className="flex items-center justify-between px-0 pt-3">
              <h4 className="text-xs font-semibold dark:text-gray-200 text-gray-800">{selected.name}</h4>
              <button onClick={() => setShowForm(true)} className="text-xs text-blue-400 hover:text-blue-300">Edit</button>
            </div>
            <GroupDetail token={token} group={selected} members={members} />
          </div>
        )}
      </div>
    </div>
  )
}

export default function Users() {
  const { token, user } = useAuth()
  // customer_owner (normal case) or super_admin (akantio staff managing
  // their own company, e.g. for support) -- matches the lm2 route gate.
  const isOwner = user?.role === 'customer_owner' || user?.role === 'super_admin'
  const effectiveToken = isOwner ? (token ?? '') : ''

  const membersState = useCompanyMembers(effectiveToken)
  const invitesState = useCompanyInvites(effectiveToken)
  const [search, setSearch] = useState('')
  const [showInvite, setShowInvite] = useState(false)
  const [showGroups, setShowGroups] = useState(false)
  const [actionError, setActionError] = useState<string | null>(null)

  if (!isOwner) {
    return (
      <div className="flex items-center justify-center h-full">
        <p className="text-sm dark:text-gray-400 text-gray-500">Only company owners can manage users.</p>
      </div>
    )
  }

  const filtered = membersState.members.filter(m =>
    fullName(m).toLowerCase().includes(search.toLowerCase()) ||
    m.email.toLowerCase().includes(search.toLowerCase()),
  )

  const handleRemove = async (m: CompanyMember) => {
    if (!window.confirm(`Remove ${fullName(m)} from this company?`)) return
    setActionError(null)
    try {
      await membersState.remove(m.membership_id)
    } catch (e) {
      setActionError(e instanceof Error ? e.message : 'Unknown error')
    }
  }

  return (
    <div className="flex flex-col h-full relative">
      <div className="flex items-center gap-3 px-5 py-3 border-b dark:border-[#2D3748] border-gray-200 shrink-0">
        <div className="relative flex-1 max-w-sm">
          <Search size={13} className="absolute left-3 top-1/2 -translate-y-1/2 dark:text-gray-500 text-gray-400" />
          <input
            value={search}
            onChange={e => setSearch(e.target.value)}
            placeholder="Search users..."
            className="w-full pl-8 pr-3 py-1.5 text-xs rounded-lg dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-200 text-gray-800 dark:placeholder-gray-500 placeholder-gray-400 outline-none focus:ring-1 focus:ring-blue-500"
          />
        </div>
        <button
          onClick={() => setShowGroups(!showGroups)}
          className={clsx(
            'flex items-center gap-1.5 px-3 py-1.5 rounded-lg border text-xs transition-colors',
            showGroups
              ? 'bg-blue-600/15 border-blue-500/30 text-blue-400'
              : 'dark:bg-[#1F2937] bg-white dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-600',
          )}
        >
          <Shield size={13} /> Groups
        </button>
        <button
          onClick={() => setShowInvite(true)}
          className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-blue-600 text-white text-xs hover:bg-blue-700 ml-auto font-medium"
        >
          <Plus size={13} /> Invite User
        </button>
      </div>

      {showInvite && (
        <InviteForm
          onCancel={() => setShowInvite(false)}
          onSubmit={(email, role) => invitesState.create(email, role)}
        />
      )}

      {(membersState.error || invitesState.error || actionError) && (
        <p className="px-5 py-2 text-xs text-red-400 border-b dark:border-[#2D3748] border-gray-200">
          {membersState.error ?? invitesState.error ?? actionError}
        </p>
      )}

      <div className="flex-1 overflow-hidden flex gap-0">
        <div className={clsx('flex flex-col overflow-hidden', showGroups ? 'flex-1' : 'w-full')}>
          <div className="flex-1 overflow-auto">
            {invitesState.invites.length > 0 && (
              <div className="px-5 py-3 border-b dark:border-[#2D3748] border-gray-200">
                <h3 className="text-xs font-semibold dark:text-gray-300 text-gray-600 uppercase tracking-wider mb-2">Pending Invites</h3>
                <ul className="space-y-1.5">
                  {invitesState.invites.map(inv => (
                    <li key={inv.id} className="flex items-center justify-between text-xs dark:text-gray-300 text-gray-700">
                      <span>{inv.email} <span className="dark:text-gray-500 text-gray-400">({ROLE_LABELS[inv.role]})</span></span>
                      <button
                        aria-label={`Revoke invite for ${inv.email}`}
                        onClick={() => invitesState.revoke(inv.id)}
                        className="text-red-400 hover:text-red-300"
                      >
                        Revoke
                      </button>
                    </li>
                  ))}
                </ul>
              </div>
            )}
            <table className="w-full text-xs">
              <thead className="sticky top-0 z-10 dark:bg-[#111827] bg-gray-50 border-b dark:border-[#2D3748] border-gray-200">
                <tr>
                  {['User', 'Role', 'Status', 'Last Login', ''].map(h => (
                    <th key={h} className="px-3 py-2.5 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider">
                      {h}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody className="divide-y dark:divide-[#2D3748]/60 divide-gray-100">
                {filtered.map(m => (
                  <tr key={m.membership_id} className="hover:dark:bg-[#1F2937]/60 hover:bg-gray-50 transition-colors group">
                    <td className="px-3 py-3">
                      <div className="flex items-center gap-2.5">
                        <div className="w-7 h-7 rounded-full bg-gradient-to-br from-blue-500 to-purple-600 flex items-center justify-center text-[10px] font-bold text-white shrink-0">
                          {fullName(m).slice(0, 2).toUpperCase()}
                        </div>
                        <div>
                          <p className="font-medium dark:text-gray-100 text-gray-900">{fullName(m)}</p>
                          <p className="text-[10px] dark:text-gray-500 text-gray-400">{m.email}</p>
                        </div>
                      </div>
                    </td>
                    <td className="px-3 py-3">
                      <label className="sr-only" htmlFor={`role-${m.membership_id}`}>Role for {fullName(m)}</label>
                      <select
                        id={`role-${m.membership_id}`}
                        aria-label={`Role for ${fullName(m)}`}
                        value={m.role}
                        onChange={e => membersState.updateRole(m.membership_id, e.target.value as MemberRole)}
                        className={clsx('px-2 py-0.5 rounded-full text-[10px] font-medium border-0 outline-none', ROLE_COLORS[m.role])}
                      >
                        <option value="customer_owner">Owner</option>
                        <option value="customer_user">User</option>
                      </select>
                    </td>
                    <td className="px-3 py-3">
                      <div className="flex items-center gap-1.5">
                        {m.status === 'active'
                          ? <CheckCircle size={12} className="text-green-400" />
                          : <XCircle size={12} className="text-gray-400" />
                        }
                        <span className={clsx('text-[10px] font-medium', m.status === 'active' ? 'text-green-400' : 'dark:text-gray-500 text-gray-400')}>
                          {m.status === 'active' ? 'Active' : m.status}
                        </span>
                      </div>
                    </td>
                    <td className="px-3 py-3 dark:text-gray-500 text-gray-400">
                      {m.last_login_at ? new Date(m.last_login_at).toLocaleDateString() : 'Never'}
                    </td>
                    <td className="px-3 py-3">
                      {m.user_id !== user?.user_id && (
                        <button
                          aria-label={`Remove ${fullName(m)}`}
                          onClick={() => handleRemove(m)}
                          className="opacity-0 group-hover:opacity-100 text-[10px] text-red-400 hover:text-red-300"
                        >
                          Remove
                        </button>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
            {!membersState.loading && filtered.length === 0 && (
              <p className="px-5 py-4 text-xs dark:text-gray-500 text-gray-400">No members found.</p>
            )}
          </div>
        </div>

        {showGroups && <GroupsPanel token={effectiveToken} members={membersState.members} />}
      </div>
    </div>
  )
}
