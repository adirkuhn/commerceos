import type {
  Group,
  CreateGroupPayload,
  UpdateGroupPayload,
  AddMemberPayload,
  AssignPermissionPayload,
  APIResponse,
  MembersResponse,
  PermissionsResponse,
} from './types'

const BASE = '/api/v1/iam'

async function request<T>(
  path: string,
  options: RequestInit,
  token: string,
): Promise<T> {
  const res = await fetch(path, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
      ...options.headers,
    },
  })
  const body: APIResponse<T> = await res.json()
  if (!res.ok || body.error) {
    throw new Error(body.error?.message ?? `HTTP ${res.status}`)
  }
  return body.data as T
}

// ── Groups ────────────────────────────────────────────────────────────────────

export function createGroup(token: string, payload: CreateGroupPayload): Promise<Group> {
  return request<Group>(`${BASE}/groups`, { method: 'POST', body: JSON.stringify(payload) }, token)
}

export function listGroups(token: string): Promise<Group[]> {
  return request<Group[]>(`${BASE}/groups`, { method: 'GET' }, token)
}

export function getGroup(token: string, id: string): Promise<Group> {
  return request<Group>(`${BASE}/groups/${id}`, { method: 'GET' }, token)
}

export function updateGroup(token: string, id: string, payload: UpdateGroupPayload): Promise<Group> {
  return request<Group>(`${BASE}/groups/${id}`, { method: 'PATCH', body: JSON.stringify(payload) }, token)
}

export function deleteGroup(token: string, id: string): Promise<null> {
  return request<null>(`${BASE}/groups/${id}`, { method: 'DELETE' }, token)
}

// ── Members ───────────────────────────────────────────────────────────────────

export function addMember(token: string, groupId: string, payload: AddMemberPayload): Promise<null> {
  return request<null>(`${BASE}/groups/${groupId}/members`, { method: 'POST', body: JSON.stringify(payload) }, token)
}

export function removeMember(token: string, groupId: string, userId: string): Promise<null> {
  return request<null>(`${BASE}/groups/${groupId}/members/${userId}`, { method: 'DELETE' }, token)
}

export function listMembers(token: string, groupId: string): Promise<MembersResponse> {
  return request<MembersResponse>(`${BASE}/groups/${groupId}/members`, { method: 'GET' }, token)
}

// ── Permissions ───────────────────────────────────────────────────────────────

export function assignPermission(token: string, groupId: string, payload: AssignPermissionPayload): Promise<null> {
  return request<null>(`${BASE}/groups/${groupId}/permissions`, { method: 'POST', body: JSON.stringify(payload) }, token)
}

export function revokePermission(token: string, groupId: string, rule: string): Promise<null> {
  return request<null>(`${BASE}/groups/${groupId}/permissions/${rule}`, { method: 'DELETE' }, token)
}

export function listGroupPermissions(token: string, groupId: string): Promise<PermissionsResponse> {
  return request<PermissionsResponse>(`${BASE}/groups/${groupId}/permissions`, { method: 'GET' }, token)
}

export function getUserEffectivePermissions(token: string, userId: string): Promise<PermissionsResponse> {
  return request<PermissionsResponse>(`${BASE}/users/${userId}/permissions`, { method: 'GET' }, token)
}
