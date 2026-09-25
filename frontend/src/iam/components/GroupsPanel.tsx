import { useState } from 'react'
import { useGroups } from '../hooks/useGroups'
import { useGroupMembers } from '../hooks/useGroupMembers'
import { useGroupPermissions } from '../hooks/useGroupPermissions'
import { GroupList } from './GroupList'
import { GroupForm } from './GroupForm'
import type { CompanyMember } from '../../team/types'
import type { Group } from '../types'
import { fullName } from '../../format/personName'

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

export function GroupsPanel({ token, members, groupId, editing, onSelect, onCreate, onEdit, onCloseForm, onDeselect }: GroupsPanelProps) {
  const groups = useGroups(token)
  const selected = groupId ? groups.groups.find(g => g.id === groupId) ?? null : null
  // Editing an existing group waits for it to load so the form opens
  // prefilled rather than as an empty "create" form.
  const showForm = editing && (groupId === null || selected !== null)

  return (
    <div className="w-[480px] border-l border-line flex flex-col overflow-hidden">
      <div className="px-4 py-3 border-b border-line shrink-0">
        <h3 className="text-sm font-semibold text-ink">Groups & permissions</h3>
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
