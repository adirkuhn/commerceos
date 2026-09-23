import { useCallback } from 'react'
import { useLocation } from 'react-router-dom'
import { subPathSegments } from '../routes'
import { useStackNavigation } from '../routing/useStackNavigation'
import { NEW_RECORD } from '../routing/useRoutedEditor'

export const USERS_PATH = '/users'
const INVITE_SEGMENT = 'invite'
const GROUPS_SEGMENT = 'groups'
const GROUPS_PATH = `${USERS_PATH}/${GROUPS_SEGMENT}`

export type UsersView =
  | { kind: 'list' }
  | { kind: 'invite' }
  | { kind: 'member'; userId: string }
  // groupId null = no group selected (or creating one); editing = the
  // group form is open (create when groupId is null).
  | { kind: 'groups'; groupId: string | null; editing: boolean }

// Which Users side panel/form is open, from the URL:
//   /users                      member list
//   /users/invite               invite form
//   /users/<userId>             that member's activity
//   /users/groups[/<id>]        groups panel, optionally a group selected
//   /users/groups/new           create-group form
//   /users/groups/<id>/edit     edit-group form
export function useUsersRoute() {
  const location = useLocation()
  const { open, exitTo, exitToOpener } = useStackNavigation()
  const [first, second, third] = subPathSegments(location.pathname, USERS_PATH)

  let view: UsersView
  if (!first) view = { kind: 'list' }
  else if (first === INVITE_SEGMENT) view = { kind: 'invite' }
  else if (first === GROUPS_SEGMENT) {
    if (second === NEW_RECORD) view = { kind: 'groups', groupId: null, editing: true }
    else view = { kind: 'groups', groupId: second ?? null, editing: !!second && third === 'edit' }
  } else view = { kind: 'member', userId: first }

  const groupPath = (id: string) => `${GROUPS_PATH}/${encodeURIComponent(id)}`
  const editingGroupId = view.kind === 'groups' && view.editing ? view.groupId : null

  return {
    view,
    openInvite: useCallback(() => open(`${USERS_PATH}/${INVITE_SEGMENT}`), [open]),
    openMember: useCallback((userId: string) => open(`${USERS_PATH}/${encodeURIComponent(userId)}`), [open]),
    openGroups: useCallback(() => open(GROUPS_PATH), [open]),
    openGroup: useCallback((id: string) => open(groupPath(id)), [open]),
    newGroup: useCallback(() => open(`${GROUPS_PATH}/${NEW_RECORD}`), [open]),
    editGroup: useCallback((id: string) => open(`${groupPath(id)}/edit`), [open]),
    closeGroupForm: useCallback(
      () => exitTo(editingGroupId ? groupPath(editingGroupId) : GROUPS_PATH),
      [exitTo, editingGroupId],
    ),
    close: useCallback(() => exitToOpener(USERS_PATH), [exitToOpener]),
  }
}
