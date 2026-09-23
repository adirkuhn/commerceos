import { useState } from 'react'
import { Plus, Search, Shield, X, CheckCircle, XCircle } from 'lucide-react'
import clsx from 'clsx'
import { useAuth } from '../auth/AuthContext'
import { useUsersRoute } from '../team/useUsersRoute'
import { useQueryString } from '../routing/useQueryState'
import { useCompanyMembers } from '../team/hooks/useCompanyMembers'
import { useCompanyInvites } from '../team/hooks/useCompanyInvites'
import { useGroups } from '../iam/hooks/useGroups'
import { useGroupMembers } from '../iam/hooks/useGroupMembers'
import { useGroupPermissions } from '../iam/hooks/useGroupPermissions'
import { useUserActivities } from '../iam/hooks/useUserActivities'
import { GroupList } from '../iam/components/GroupList'
import { GroupForm } from '../iam/components/GroupForm'
import type { MemberRole, CompanyMember } from '../team/types'
import type { Group } from '../iam/types'
import { useConfirm } from '../components/common/useConfirm'

const ROLE_LABELS: Record<MemberRole, string> = { customer_owner: 'Owner', customer_user: 'User' }
const ROLE_COLORS: Record<MemberRole, string> = {
  customer_owner: 'bg-red-500/15 text-danger',
  customer_user: 'bg-gray-500/15 text-ink-4',
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
    <div className="absolute right-5 top-14 z-20 w-80 bg-raised rounded-xl border border-line shadow-lg p-4">
      <div className="flex items-center justify-between mb-3">
        <h3 className="text-sm font-semibold text-ink">Invite User</h3>
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
        <h4 className="text-xs font-semibold text-ink-2 uppercase tracking-wider mb-2">Members</h4>
        {groupMembers.error && <p className="text-xs text-danger mb-2">{groupMembers.error}</p>}
        <ul className="space-y-1 mb-3">
          {groupMembers.members.map(userId => (
            <li key={userId} className="flex items-center justify-between text-xs text-ink-2">
              <span>{memberByID.get(userId)?.email ?? userId}</span>
              <button onClick={() => groupMembers.remove(userId)} className="text-danger hover:text-red-300">Remove</button>
            </li>
          ))}
          {groupMembers.members.length === 0 && <li className="text-xs text-ink-4">No members yet.</li>}
        </ul>
        <div className="flex gap-2">
          <select
            aria-label="Add member to group"
            value={pickUserId}
            onChange={e => setPickUserId(e.target.value)}
            className="flex-1 px-2 py-1 text-xs rounded border border-line bg-sunken text-ink"
          >
            <option value="">Select a member…</option>
            {notInGroup.map(m => (
              <option key={m.user_id} value={m.user_id}>{fullName(m)}</option>
            ))}
          </select>
          <button
            onClick={() => { if (pickUserId) { groupMembers.add(pickUserId); setPickUserId('') } }}
            disabled={!pickUserId}
            className="px-3 py-1 text-xs rounded bg-accent-solid text-white disabled:opacity-50"
          >
            Add
          </button>
        </div>
      </div>

      <div>
        <h4 className="text-xs font-semibold text-ink-2 uppercase tracking-wider mb-2">Permissions</h4>
        {groupPerms.error && <p className="text-xs text-danger mb-2">{groupPerms.error}</p>}
        <ul className="space-y-1 mb-3">
          {groupPerms.permissions.map(rule => (
            <li key={rule} className="flex items-center justify-between text-xs font-mono text-ink-2">
              <span>{rule}</span>
              <button onClick={() => groupPerms.revoke(rule)} className="text-danger hover:text-red-300 font-sans">Revoke</button>
            </li>
          ))}
          {groupPerms.permissions.length === 0 && <li className="text-xs text-ink-4">No permissions yet.</li>}
        </ul>
        <div className="flex gap-2">
          <input
            aria-label="New permission rule"
            placeholder="pms.product.create"
            value={newRule}
            onChange={e => setNewRule(e.target.value)}
            className="flex-1 px-2 py-1 text-xs rounded border border-line bg-sunken text-ink font-mono"
          />
          <button
            onClick={() => { if (newRule) { groupPerms.assign(newRule); setNewRule('') } }}
            disabled={!newRule}
            className="px-3 py-1 text-xs rounded bg-accent-solid text-white disabled:opacity-50"
          >
            Assign
          </button>
        </div>
      </div>
    </div>
  )
}

function UserActivityPanel({ token, member, onClose }: { token: string; member: CompanyMember; onClose: () => void }) {
  const { activities, loading, error } = useUserActivities(token, member.user_id)

  return (
    <div className="w-[480px] border-l border-line flex flex-col overflow-hidden">
      <div className="px-4 py-3 border-b border-line shrink-0 flex items-center justify-between">
        <div>
          <h3 className="text-sm font-semibold text-ink">{fullName(member)}</h3>
          <p className="text-2xs text-ink-4 mt-0.5">Activity across all their orders</p>
        </div>
        <button onClick={onClose} aria-label="Close" className="p-1 -m-1 rounded hover:bg-hover"><X size={14} className="text-ink-3" /></button>
      </div>
      <div className="flex-1 overflow-auto p-4 space-y-2">
        {error && <p className="text-xs text-danger">{error}</p>}
        {!loading && activities.length === 0 && !error && (
          <p className="text-xs text-ink-4">No activity yet.</p>
        )}
        {activities.map(a => (
          <div key={a.id} className="flex justify-between gap-3 text-xs">
            <span className="text-ink">{a.summary}</span>
            <span className="shrink-0 text-ink-4">{new Date(a.created_at).toLocaleString()}</span>
          </div>
        ))}
      </div>
    </div>
  )
}

interface GroupsPanelProps {
  token: string
  members: CompanyMember[]
  // From the URL (see useUsersRoute): the selected group, and whether the
  // group form is open (create when groupId is null).
  groupId: string | null
  editing: boolean
  onSelect: (id: string) => void
  onCreate: () => void
  onEdit: (id: string) => void
  onCloseForm: () => void
  onDeselect: () => void
}

function GroupsPanel({ token, members, groupId, editing, onSelect, onCreate, onEdit, onCloseForm, onDeselect }: GroupsPanelProps) {
  const groups = useGroups(token)
  const selected = groupId ? groups.groups.find(g => g.id === groupId) ?? null : null
  // Editing an existing group waits for it to load so the form opens
  // prefilled rather than as an empty "create" form.
  const showForm = editing && (groupId === null || selected !== null)

  return (
    <div className="w-[480px] border-l border-line flex flex-col overflow-hidden">
      <div className="px-4 py-3 border-b border-line shrink-0">
        <h3 className="text-sm font-semibold text-ink">Groups & Permissions</h3>
        <p className="text-2xs text-ink-4 mt-0.5">Fine-grained access on top of owner/user roles</p>
      </div>
      <div className="flex-1 overflow-auto p-4">
        {showForm ? (
          <GroupForm
            key={groupId ?? 'new'}
            group={selected}
            onCancel={onCloseForm}
            onSubmit={async payload => {
              if (selected) await groups.update(selected.id, payload)
              else await groups.create(payload)
              onCloseForm()
            }}
          />
        ) : (
          <GroupList
            groups={groups.groups}
            loading={groups.loading}
            error={groups.error}
            onSelect={g => onSelect(g.id)}
            onDelete={id => { groups.remove(id); if (groupId === id) onDeselect() }}
            onCreateClick={onCreate}
          />
        )}
        {selected && !showForm && (
          <div className="mt-4 border-t border-line">
            <div className="flex items-center justify-between px-0 pt-3">
              <h4 className="text-xs font-semibold text-ink">{selected.name}</h4>
              <button onClick={() => onEdit(selected.id)} className="text-xs text-accent hover:text-accent/80">Edit</button>
            </div>
            <GroupDetail token={token} group={selected} members={members} />
          </div>
        )}
      </div>
    </div>
  )
}

export default function Users() {
  const { confirm, confirmDialog } = useConfirm()
  const { token, user } = useAuth()
  // customer_owner (normal case) or super_admin (akantio staff managing
  // their own company, e.g. for support) -- matches the lm2 route gate.
  const isOwner = user?.role === 'customer_owner' || user?.role === 'super_admin'
  const effectiveToken = isOwner ? (token ?? '') : ''

  const membersState = useCompanyMembers(effectiveToken)
  const invitesState = useCompanyInvites(effectiveToken)
  // Search (?q=) and the open panel/form (see useUsersRoute) live in the
  // URL, so refresh and Back keep the user where they were.
  const [searchParam, setSearch] = useQueryString('q')
  const search = searchParam ?? ''
  const route = useUsersRoute()
  const { view } = route
  const showInvite = view.kind === 'invite'
  const showGroups = view.kind === 'groups'
  const selectedUser = view.kind === 'member'
    ? membersState.members.find(m => m.user_id === view.userId) ?? null
    : null
  const [actionError, setActionError] = useState<string | null>(null)

  if (!isOwner) {
    return (
      <div className="flex items-center justify-center h-full">
        <p className="text-sm text-ink-3">Only company owners can manage users.</p>
      </div>
    )
  }

  const filtered = membersState.members.filter(m =>
    fullName(m).toLowerCase().includes(search.toLowerCase()) ||
    m.email.toLowerCase().includes(search.toLowerCase()),
  )

  const handleRemove = async (m: CompanyMember) => {
    if (!(await confirm({ title: `Remove ${fullName(m)} from this company?`, message: 'They lose access to this company right away.', confirmLabel: 'Remove member', destructive: true }))) return
    setActionError(null)
    try {
      await membersState.remove(m.membership_id)
    } catch (e) {
      setActionError(e instanceof Error ? e.message : 'Unknown error')
    }
  }

  return (
    <div className="flex flex-col h-full relative">
      <div className="flex items-center gap-3 px-5 py-3 border-b border-line shrink-0">
        <div className="relative flex-1 max-w-sm">
          <Search size={13} className="absolute left-3 top-1/2 -translate-y-1/2 text-ink-4" />
          <input
            value={search}
            onChange={e => setSearch(e.target.value)}
            placeholder="Search users..."
            className="w-full pl-8 pr-3 py-1.5 text-xs rounded-lg bg-raised border border-line text-ink placeholder-ink-4 outline-none focus:ring-1 focus:ring-focus"
          />
        </div>
        <button
          onClick={() => (showGroups ? route.close() : route.openGroups())}
          className={clsx(
            'flex items-center gap-1.5 px-3 py-1.5 rounded-lg border text-xs transition-colors',
            showGroups
              ? 'bg-accent/15 border-accent/30 text-accent'
              : 'bg-raised border-line text-ink-2',
          )}
        >
          <Shield size={13} /> Groups
        </button>
        <button
          onClick={route.openInvite}
          className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-accent-solid text-white text-xs hover:bg-accent-solid-hover ml-auto font-medium"
        >
          <Plus size={13} /> Invite User
        </button>
      </div>

      {showInvite && (
        <InviteForm
          onCancel={route.close}
          onSubmit={(email, role) => invitesState.create(email, role)}
        />
      )}

      {(membersState.error || invitesState.error || actionError) && (
        <p className="px-5 py-2 text-xs text-danger border-b border-line">
          {membersState.error ?? invitesState.error ?? actionError}
        </p>
      )}

      <div className="flex-1 overflow-hidden flex gap-0">
        <div className={clsx('flex flex-col overflow-hidden', (showGroups || selectedUser) ? 'flex-1' : 'w-full')}>
          <div className="flex-1 overflow-auto">
            {invitesState.invites.length > 0 && (
              <div className="px-5 py-3 border-b border-line">
                <h3 className="text-xs font-semibold text-ink-2 uppercase tracking-wider mb-2">Pending Invites</h3>
                <ul className="space-y-1.5">
                  {invitesState.invites.map(inv => (
                    <li key={inv.id} className="flex items-center justify-between text-xs text-ink-2">
                      <span>{inv.email} <span className="text-ink-4">({ROLE_LABELS[inv.role]})</span></span>
                      <button
                        aria-label={`Revoke invite for ${inv.email}`}
                        onClick={() => invitesState.revoke(inv.id)}
                        className="text-danger hover:text-red-300"
                      >
                        Revoke
                      </button>
                    </li>
                  ))}
                </ul>
              </div>
            )}
            <table className="w-full text-xs">
              <thead className="sticky top-0 z-10 bg-sunken border-b border-line">
                <tr>
                  {['User', 'Role', 'Status', 'Last Login', ''].map(h => (
                    <th key={h} className="px-3 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">
                      {h}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody className="divide-y divide-line/60">
                {filtered.map(m => (
                  <tr key={m.membership_id} className="hover:bg-hover/60 transition-colors group">
                    <td className="px-3 py-3">
                      <button
                        onClick={() => route.openMember(m.user_id)}
                        className="flex items-center gap-2.5 text-left hover:underline"
                        title={`View ${fullName(m)}'s activity`}
                      >
                        <div className="w-7 h-7 rounded-full bg-gradient-to-br from-blue-500 to-purple-600 flex items-center justify-center text-2xs font-bold text-white shrink-0">
                          {fullName(m).slice(0, 2).toUpperCase()}
                        </div>
                        <div>
                          <p className="font-medium text-ink">{fullName(m)}</p>
                          <p className="text-2xs text-ink-4">{m.email}</p>
                        </div>
                      </button>
                    </td>
                    <td className="px-3 py-3">
                      <label className="sr-only" htmlFor={`role-${m.membership_id}`}>Role for {fullName(m)}</label>
                      <select
                        id={`role-${m.membership_id}`}
                        aria-label={`Role for ${fullName(m)}`}
                        value={m.role}
                        onChange={e => membersState.updateRole(m.membership_id, e.target.value as MemberRole)}
                        className={clsx('px-2 py-0.5 rounded-full text-2xs font-medium border-0 outline-none focus-visible:ring-2 focus-visible:ring-focus/60', ROLE_COLORS[m.role])}
                      >
                        <option value="customer_owner">Owner</option>
                        <option value="customer_user">User</option>
                      </select>
                    </td>
                    <td className="px-3 py-3">
                      <div className="flex items-center gap-1.5">
                        {m.status === 'active'
                          ? <CheckCircle size={12} className="text-success" />
                          : <XCircle size={12} className="text-ink-4" />
                        }
                        <span className={clsx('text-2xs font-medium', m.status === 'active' ? 'text-success' : 'text-ink-4')}>
                          {m.status === 'active' ? 'Active' : m.status}
                        </span>
                      </div>
                    </td>
                    <td className="px-3 py-3 text-ink-4">
                      {m.last_login_at ? new Date(m.last_login_at).toLocaleDateString() : 'Never'}
                    </td>
                    <td className="px-3 py-3">
                      {m.user_id !== user?.user_id && (
                        <button
                          aria-label={`Remove ${fullName(m)}`}
                          onClick={() => handleRemove(m)}
                          className="opacity-0 group-hover:opacity-100 focus-within:opacity-100 text-2xs text-danger hover:text-red-300"
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
              <p className="px-5 py-4 text-xs text-ink-4">No members found.</p>
            )}
          </div>
        </div>

        {view.kind === 'groups' && (
          <GroupsPanel
            token={effectiveToken}
            members={membersState.members}
            groupId={view.groupId}
            editing={view.editing}
            onSelect={route.openGroup}
            onCreate={route.newGroup}
            onEdit={route.editGroup}
            onCloseForm={route.closeGroupForm}
            onDeselect={route.openGroups}
          />
        )}
        {selectedUser && <UserActivityPanel token={effectiveToken} member={selectedUser} onClose={route.close} />}
      </div>
      {confirmDialog}
    </div>
  )
}
