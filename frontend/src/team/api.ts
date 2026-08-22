import { APP_ID, lm2Origin } from '../auth/loginConfig'
import type { APIResponse, CompanyMember, CompanyInvite, MemberRole } from './types'

function base(): string {
  return `${lm2Origin()}/api/v1/app`
}

async function request<T>(path: string, token: string, options: RequestInit = {}): Promise<T> {
  const res = await fetch(path, {
    ...options,
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}`, ...options.headers },
  })
  const body = await res.json()
  if (!res.ok) {
    throw new Error(body.message ?? body.error ?? `HTTP ${res.status}`)
  }
  return (body as APIResponse<T>).data
}

export async function listMembers(token: string): Promise<CompanyMember[]> {
  const result = await request<{ items: CompanyMember[]; total: number }>(`${base()}/company/members`, token)
  return result.items
}

export function updateMemberRole(token: string, membershipId: string, role: MemberRole): Promise<null> {
  return request<null>(`${base()}/company/members/${membershipId}`, token, {
    method: 'PATCH', body: JSON.stringify({ role }),
  })
}

export function removeMember(token: string, membershipId: string): Promise<null> {
  return request<null>(`${base()}/company/members/${membershipId}`, token, { method: 'DELETE' })
}

export async function listInvites(token: string): Promise<CompanyInvite[]> {
  const result = await request<{ items: CompanyInvite[]; total: number }>(`${base()}/company/invites`, token)
  return result.items
}

export function createInvite(token: string, email: string, role: MemberRole): Promise<{ id: string; email: string; role: MemberRole }> {
  return request(`${base()}/company/invites`, token, {
    method: 'POST',
    body: JSON.stringify({
      email, role, app: APP_ID,
      accept_base: `${window.location.origin}/accept-invite`,
    }),
  })
}

export function revokeInvite(token: string, inviteId: string): Promise<null> {
  return request<null>(`${base()}/company/invites/${inviteId}`, token, { method: 'DELETE' })
}

export interface AcceptInviteInput {
  token: string
  firstName?: string
  lastName?: string
  password?: string
}

// acceptInvite is public -- the invitee has no session yet, so this hits
// lm2 directly without an Authorization header (unlike request() above).
export async function acceptInvite(input: AcceptInviteInput): Promise<null> {
  const res = await fetch(`${lm2Origin()}/api/v1/auth/app/invites/accept`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      token: input.token, first_name: input.firstName, last_name: input.lastName, password: input.password,
    }),
  })
  const body = await res.json()
  if (!res.ok) {
    throw new Error(body.message ?? body.error ?? `HTTP ${res.status}`)
  }
  return body.data as null
}
