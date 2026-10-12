import { useState } from 'react'
import { Plus, Search, Shield, X, CheckCircle, XCircle } from 'lucide-react'
import clsx from 'clsx'
import { useTranslation } from 'react-i18next'
import { activityText } from '../oms/activityText'
import type { TFunction } from 'i18next'
import { useAuth } from '../auth/AuthContext'
import { useUsersRoute } from '../team/useUsersRoute'
import { useQueryString } from '../routing/useQueryState'
import { useCompanyMembers } from '../team/hooks/useCompanyMembers'
import { useCompanyInvites } from '../team/hooks/useCompanyInvites'
import { useUserActivities } from '../iam/hooks/useUserActivities'
import type { MemberRole, CompanyMember } from '../team/types'
import { useConfirm } from '../components/common/useConfirm'
import { GroupsPanel } from '../iam/components/GroupsPanel'
import { InviteForm } from '../iam/components/InviteForm'
import { fullName } from '../format/personName'
import { useDateFormat } from '../format/DateFormatProvider'
import { apiErrorText } from '../api/apiErrorText'
import { LIST_TOOLBAR_SINGLE } from '../components/common/listToolbar'

function roleLabel(t: TFunction<['iam', 'common']>, role: MemberRole): string {
  return role === 'customer_owner' ? t('users.roleOwner') : t('users.roleUser')
}
const ROLE_COLORS: Record<MemberRole, string> = {
  customer_owner: 'bg-red-500/15 text-danger',
  customer_user: 'bg-gray-500/15 text-ink-4',
}

function UserActivityPanel({ token, member, onClose }: { token: string; member: CompanyMember; onClose: () => void }) {
  const { formatDateTime } = useDateFormat()
  const { t } = useTranslation(['iam', 'common'])
  const { activities, loading, error } = useUserActivities(token, member.user_id)
  const { t: tOrders } = useTranslation('orders')

  return (
    <div className="w-[480px] border-l border-line flex flex-col overflow-hidden">
      <div className="px-4 py-3 border-b border-line shrink-0 flex items-center justify-between">
        <div>
          <h3 className="text-sm font-semibold text-ink">{fullName(member)}</h3>
          <p className="text-2xs text-ink-4 mt-0.5">{t('users.activitySubtitle')}</p>
        </div>
        <button onClick={onClose} aria-label={t('common:actions.close')} className="p-1 -m-1 rounded hover:bg-hover"><X size={14} className="text-ink-3" /></button>
      </div>
      <div className="flex-1 overflow-auto p-4 space-y-2">
        {error && <p className="text-xs text-danger">{error}</p>}
        {!loading && activities.length === 0 && !error && (
          <p className="text-xs text-ink-4">{t('users.noActivity')}</p>
        )}
        {activities.map(a => (
          <div key={a.id} className="flex justify-between gap-3 text-xs">
            <span className="text-ink">{activityText(tOrders, a)}</span>
            <span className="shrink-0 text-ink-4">{formatDateTime(a.created_at)}</span>
          </div>
        ))}
      </div>
    </div>
  )
}

export default function Users() {
  const { formatDate } = useDateFormat()
  const { t } = useTranslation(['iam', 'common'])
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
        <p className="text-sm text-ink-3">{t('users.ownersOnly')}</p>
      </div>
    )
  }

  const filtered = membersState.members.filter(m =>
    fullName(m).toLowerCase().includes(search.toLowerCase()) ||
    m.email.toLowerCase().includes(search.toLowerCase()),
  )

  const handleRemove = async (m: CompanyMember) => {
    if (!(await confirm({ title: t('users.removeTitle', { name: fullName(m) }), message: t('users.removeMessage'), confirmLabel: t('users.removeConfirm'), destructive: true }))) return
    setActionError(null)
    try {
      await membersState.remove(m.membership_id)
    } catch (e) {
      setActionError(apiErrorText(e) ?? t('unknownError'))
    }
  }

  return (
    <div className="flex flex-col h-full relative">
      <div className={clsx('flex items-center gap-3 px-5 py-3 border-b border-line shrink-0', LIST_TOOLBAR_SINGLE)}>
        <div className="relative flex-1 max-w-sm">
          <Search size={13} className="absolute left-3 top-1/2 -translate-y-1/2 text-ink-4" />
          <input
            value={search}
            onChange={e => setSearch(e.target.value)}
            placeholder={t('users.search')}
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
          <Shield size={13} /> {t('groups.title')}
        </button>
        <button
          onClick={route.openInvite}
          className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-accent-solid text-white text-xs hover:bg-accent-solid-hover ml-auto font-medium"
        >
          <Plus size={13} /> {t('users.invite')}
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
                <h3 className="text-xs font-semibold text-ink-2 caps-label mb-2">{t('users.pendingInvites')}</h3>
                <ul className="space-y-1.5">
                  {invitesState.invites.map(inv => (
                    <li key={inv.id} className="flex items-center justify-between text-xs text-ink-2">
                      <span>{inv.email} <span className="text-ink-4">({roleLabel(t, inv.role)})</span></span>
                      <button
                        aria-label={t('users.revokeInvite', { email: inv.email })}
                        onClick={() => invitesState.revoke(inv.id)}
                        className="text-danger hover:text-red-300"
                      >
                        {t('groups.revoke')}
                      </button>
                    </li>
                  ))}
                </ul>
              </div>
            )}
            <table className="w-full text-xs">
              <thead className="sticky top-0 z-10 bg-sunken border-b border-line">
                <tr>
                  {[t('users.colUser'), t('users.role'), t('common:fields.status'), t('users.colLastLogin'), ''].map(h => (
                    <th key={h} className="px-3 py-2.5 text-left font-medium text-ink-4 caps-label">
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
                        title={t('users.viewActivity', { name: fullName(m) })}
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
                      <label className="sr-only" htmlFor={`role-${m.membership_id}`}>{t('users.roleFor', { name: fullName(m) })}</label>
                      <select
                        id={`role-${m.membership_id}`}
                        aria-label={t('users.roleFor', { name: fullName(m) })}
                        value={m.role}
                        onChange={e => membersState.updateRole(m.membership_id, e.target.value as MemberRole)}
                        className={clsx('px-2 py-0.5 rounded-full text-2xs font-medium border-0 outline-none focus-visible:ring-2 focus-visible:ring-focus/60', ROLE_COLORS[m.role])}
                      >
                        <option value="customer_owner">{t('users.roleOwner')}</option>
                        <option value="customer_user">{t('users.roleUser')}</option>
                      </select>
                    </td>
                    <td className="px-3 py-3">
                      <div className="flex items-center gap-1.5">
                        {m.status === 'active'
                          ? <CheckCircle size={12} className="text-success" />
                          : <XCircle size={12} className="text-ink-4" />
                        }
                        <span className={clsx('text-2xs font-medium', m.status === 'active' ? 'text-success' : 'text-ink-4')}>
                          {m.status === 'active' ? t('common:status.active') : m.status === 'inactive' ? t('common:status.inactive') : m.status}
                        </span>
                      </div>
                    </td>
                    <td className="px-3 py-3 text-ink-4">
                      {m.last_login_at ? formatDate(m.last_login_at) : t('users.never')}
                    </td>
                    <td className="px-3 py-3">
                      {m.user_id !== user?.user_id && (
                        <button
                          aria-label={t('users.removeLabel', { name: fullName(m) })}
                          onClick={() => handleRemove(m)}
                          className="opacity-0 group-hover:opacity-100 focus-within:opacity-100 text-2xs text-danger hover:text-red-300"
                        >
                          {t('common:actions.remove')}
                        </button>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
            {!membersState.loading && filtered.length === 0 && (
              <p className="px-5 py-4 text-xs text-ink-4">{t('users.noneFound')}</p>
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
