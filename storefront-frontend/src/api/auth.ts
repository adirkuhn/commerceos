import { request } from './client'
import type { Account, AccountOrder, Session } from '../types'

export function requestMagicLink(email: string, redirectUrl: string): Promise<null> {
  return request<null>('/auth/magic-link', {
    method: 'POST',
    body: JSON.stringify({ email, redirect_url: redirectUrl }),
  })
}

export function verifySession(token: string): Promise<Session> {
  return request<Session>('/auth/session', {
    method: 'POST',
    body: JSON.stringify({ token }),
  })
}

export function getAccount(sessionToken: string): Promise<Account> {
  return request<Account>(`/account/${encodeURIComponent(sessionToken)}`)
}

export function listAccountOrders(sessionToken: string): Promise<AccountOrder[]> {
  return request<AccountOrder[]>(`/account/${encodeURIComponent(sessionToken)}/orders`)
}

export function logout(sessionToken: string): Promise<null> {
  return request<null>(`/account/${encodeURIComponent(sessionToken)}`, { method: 'DELETE' })
}
