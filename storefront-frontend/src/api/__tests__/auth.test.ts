import { describe, it, expect, vi, beforeEach } from 'vitest'
import { requestMagicLink, verifySession, getAccount, listAccountOrders, logout } from '../auth'
import type { APIResponse, Session, Account, AccountOrder } from '../../types'

function mockFetch<T>(data: T) {
  const body: APIResponse<T> = { data, error: null }
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, status: 200, json: () => Promise.resolve(body) }))
}

function mockFetchError(code: string, message: string, status = 422) {
  const body: APIResponse<null> = { data: null, error: { code, message } }
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false, status, json: () => Promise.resolve(body) }))
}

beforeEach(() => { vi.restoreAllMocks() })

describe('auth api', () => {
  it('requestMagicLink POSTs email and redirect_url', async () => {
    mockFetch(null)
    await requestMagicLink('shopper@example.com', 'https://shop.example.com/login')
    const [url, options] = vi.mocked(fetch).mock.calls[0]
    expect(url).toContain('/storefront/v1/auth/magic-link')
    expect(options?.method).toBe('POST')
    expect(JSON.parse(options?.body as string)).toEqual({ email: 'shopper@example.com', redirect_url: 'https://shop.example.com/login' })
  })

  it('verifySession POSTs the token and returns a session', async () => {
    const session: Session = { session_token: 'session-1', account: { email: 'shopper@example.com', first_name: 'Ada', last_name: '' } }
    mockFetch(session)
    const result = await verifySession('LINK123')
    expect(result).toEqual(session)
    const [url, options] = vi.mocked(fetch).mock.calls[0]
    expect(url).toContain('/auth/session')
    expect(JSON.parse(options?.body as string)).toEqual({ token: 'LINK123' })
  })

  it('verifySession throws on an invalid token', async () => {
    mockFetchError('VALIDATION_ERROR', 'this sign-in link is invalid or has expired')
    await expect(verifySession('bad-token')).rejects.toThrow('this sign-in link is invalid or has expired')
  })

  it('getAccount fetches the profile by session token', async () => {
    const account: Account = { email: 'shopper@example.com', first_name: 'Ada', last_name: 'Lovelace' }
    mockFetch(account)
    const result = await getAccount('session-1')
    expect(result).toEqual(account)
    const [url] = vi.mocked(fetch).mock.calls[0]
    expect(url).toContain('/account/session-1')
  })

  it('listAccountOrders fetches order history by session token', async () => {
    const orders: AccountOrder[] = [{ order_number: 'WEB-1', status: 'completed', fulfilled: true, total_minor_units: 1000, currency_code: 'EUR', placed_at: '2026-06-01T00:00:00Z' }]
    mockFetch(orders)
    const result = await listAccountOrders('session-1')
    expect(result).toEqual(orders)
    const [url] = vi.mocked(fetch).mock.calls[0]
    expect(url).toContain('/account/session-1/orders')
  })

  it('logout DELETEs the session', async () => {
    mockFetch(null)
    await logout('session-1')
    const [url, options] = vi.mocked(fetch).mock.calls[0]
    expect(url).toContain('/account/session-1')
    expect(options?.method).toBe('DELETE')
  })
})
