// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach } from 'vitest'
import * as api from '../api'
import type { CompanyMember, CompanyInvite } from '../types'

const TOKEN = 'test-token'

function mockFetch<T>(data: T, status = 200) {
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
    ok: status >= 200 && status < 300,
    status,
    json: () => Promise.resolve({ data }),
  }))
}

function mockFetchError(message: string, status = 400) {
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
    ok: false,
    status,
    json: () => Promise.resolve({ error: 'bad_request', message }),
  }))
}

const sampleMember: CompanyMember = {
  membership_id: 'm-1', user_id: 'u-1', email: 'alex@example.com', first_name: 'Alex', last_name: 'Kuhn',
  role: 'customer_owner', status: 'active', last_login_at: '2026-08-20T00:00:00Z',
}

const sampleInvite: CompanyInvite = {
  id: 'inv-1', email: 'new@example.com', role: 'customer_user',
  expires_at: '2026-08-29T00:00:00Z', created_at: '2026-08-22T00:00:00Z',
}

beforeEach(() => {
  vi.restoreAllMocks()
  vi.stubEnv('VITE_LM2_LOGIN_URL', 'https://lm2.example.com/login')
})

describe('listMembers', () => {
  it('returns members from the lm2 origin', async () => {
    mockFetch({ items: [sampleMember], total: 1 })
    const result = await api.listMembers(TOKEN)
    expect(result).toEqual([sampleMember])
    const [url] = vi.mocked(fetch).mock.calls[0]
    expect(url).toBe('https://lm2.example.com/api/v1/app/company/members')
  })
})

describe('updateMemberRole', () => {
  it('PATCHes the membership role', async () => {
    mockFetch(null)
    await api.updateMemberRole(TOKEN, 'm-1', 'customer_user')
    const [url, options] = vi.mocked(fetch).mock.calls[0]
    expect(url).toBe('https://lm2.example.com/api/v1/app/company/members/m-1')
    expect(options?.method).toBe('PATCH')
    expect(JSON.parse(options?.body as string)).toEqual({ role: 'customer_user' })
  })
})

describe('removeMember', () => {
  it('DELETEs the membership', async () => {
    mockFetch(null)
    await api.removeMember(TOKEN, 'm-1')
    const [url, options] = vi.mocked(fetch).mock.calls[0]
    expect(url).toBe('https://lm2.example.com/api/v1/app/company/members/m-1')
    expect(options?.method).toBe('DELETE')
  })

  it('throws the last-owner guard message', async () => {
    mockFetchError('company must have at least one owner')
    await expect(api.removeMember(TOKEN, 'm-1')).rejects.toThrow('company must have at least one owner')
  })
})

describe('listInvites', () => {
  it('returns pending invites', async () => {
    mockFetch({ items: [sampleInvite], total: 1 })
    const result = await api.listInvites(TOKEN)
    expect(result).toEqual([sampleInvite])
  })
})

describe('createInvite', () => {
  it('sends email, role, app, and accept_base', async () => {
    mockFetch({ id: 'inv-1', email: 'new@example.com', role: 'customer_user' })
    await api.createInvite(TOKEN, 'new@example.com', 'customer_user')
    const [url, options] = vi.mocked(fetch).mock.calls[0]
    expect(url).toBe('https://lm2.example.com/api/v1/app/company/invites')
    const body = JSON.parse(options?.body as string)
    expect(body.email).toBe('new@example.com')
    expect(body.role).toBe('customer_user')
    expect(body.app).toBe('commerceos')
    expect(body.accept_base).toContain('/accept-invite')
  })
})

describe('revokeInvite', () => {
  it('DELETEs the invite', async () => {
    mockFetch(null)
    await api.revokeInvite(TOKEN, 'inv-1')
    const [url, options] = vi.mocked(fetch).mock.calls[0]
    expect(url).toBe('https://lm2.example.com/api/v1/app/company/invites/inv-1')
    expect(options?.method).toBe('DELETE')
  })
})

describe('acceptInvite', () => {
  it('POSTs without a bearer token (public endpoint)', async () => {
    mockFetch(null)
    await api.acceptInvite({ token: 'raw-token', firstName: 'Bo', lastName: 'Lee', password: 'secret123' })
    const [url, options] = vi.mocked(fetch).mock.calls[0]
    expect(url).toBe('https://lm2.example.com/api/v1/auth/app/invites/accept')
    expect((options?.headers as Record<string, string> | undefined)?.Authorization).toBeUndefined()
    const body = JSON.parse(options?.body as string)
    expect(body).toEqual({ token: 'raw-token', first_name: 'Bo', last_name: 'Lee', password: 'secret123' })
  })

  it('throws on an invalid/expired token', async () => {
    mockFetchError('invite is invalid, expired, or already used')
    await expect(api.acceptInvite({ token: 'bad' })).rejects.toThrow('invite is invalid, expired, or already used')
  })
})
