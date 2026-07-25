import { describe, it, expect, vi, beforeEach } from 'vitest'
import * as api from '../api'
import type { Group, APIResponse } from '../types'

const TOKEN = 'test-token'

function mockFetch<T>(data: T, status = 200) {
  const body: APIResponse<T> = { data, error: null }
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
    ok: status >= 200 && status < 300,
    status,
    json: () => Promise.resolve(body),
  }))
}

function mockFetchError(code: string, message: string, status = 422) {
  const body: APIResponse<null> = { data: null, error: { code, message } }
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
    ok: false,
    status,
    json: () => Promise.resolve(body),
  }))
}

const sampleGroup: Group = {
  id: 'g1',
  tenant_id: 't1',
  name: 'Editors',
  description: 'Edit stuff',
  created_at: '2026-06-13T00:00:00Z',
  updated_at: '2026-06-13T00:00:00Z',
}

beforeEach(() => { vi.restoreAllMocks() })

// ── createGroup ───────────────────────────────────────────────────────────────

describe('createGroup', () => {
  it('returns group on success', async () => {
    mockFetch(sampleGroup, 201)
    const result = await api.createGroup(TOKEN, { name: 'Editors' })
    expect(result).toEqual(sampleGroup)
  })

  it('throws on error response', async () => {
    mockFetchError('CONFLICT', 'group name already exists', 409)
    await expect(api.createGroup(TOKEN, { name: 'Editors' })).rejects.toThrow('group name already exists')
  })

  it('sends correct method and path', async () => {
    mockFetch(sampleGroup, 201)
    await api.createGroup(TOKEN, { name: 'Editors', description: 'desc' })
    const fetchMock = vi.mocked(fetch)
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/v1/iam/groups',
      expect.objectContaining({ method: 'POST' }),
    )
  })

  it('sends Authorization header', async () => {
    mockFetch(sampleGroup, 201)
    await api.createGroup(TOKEN, { name: 'Editors' })
    const fetchMock = vi.mocked(fetch)
    const [, opts] = fetchMock.mock.calls[0] as [string, RequestInit]
    expect((opts.headers as Record<string, string>)['Authorization']).toBe(`Bearer ${TOKEN}`)
  })
})

// ── listGroups ────────────────────────────────────────────────────────────────

describe('listGroups', () => {
  it('returns array of groups', async () => {
    mockFetch([sampleGroup])
    const result = await api.listGroups(TOKEN)
    expect(result).toHaveLength(1)
    expect(result[0].id).toBe('g1')
  })
})

// ── getGroup ──────────────────────────────────────────────────────────────────

describe('getGroup', () => {
  it('returns group by id', async () => {
    mockFetch(sampleGroup)
    const result = await api.getGroup(TOKEN, 'g1')
    expect(result.id).toBe('g1')
  })

  it('throws on 404', async () => {
    mockFetchError('NOT_FOUND', 'group not found', 404)
    await expect(api.getGroup(TOKEN, 'missing')).rejects.toThrow('group not found')
  })
})

// ── updateGroup ───────────────────────────────────────────────────────────────

describe('updateGroup', () => {
  it('returns updated group', async () => {
    const updated = { ...sampleGroup, name: 'Writers' }
    mockFetch(updated)
    const result = await api.updateGroup(TOKEN, 'g1', { name: 'Writers' })
    expect(result.name).toBe('Writers')
  })
})

// ── deleteGroup ───────────────────────────────────────────────────────────────

describe('deleteGroup', () => {
  it('resolves on success', async () => {
    mockFetch(null)
    await expect(api.deleteGroup(TOKEN, 'g1')).resolves.toBeNull()
  })
})

// ── members ───────────────────────────────────────────────────────────────────

describe('listMembers', () => {
  it('returns members response', async () => {
    mockFetch({ members: ['user-1', 'user-2'] })
    const result = await api.listMembers(TOKEN, 'g1')
    expect(result.members).toEqual(['user-1', 'user-2'])
  })
})

describe('addMember', () => {
  it('calls correct endpoint', async () => {
    mockFetch(null)
    await api.addMember(TOKEN, 'g1', { user_id: 'user-42' })
    const fetchMock = vi.mocked(fetch)
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/v1/iam/groups/g1/members',
      expect.objectContaining({ method: 'POST' }),
    )
  })

  it('throws on conflict', async () => {
    mockFetchError('CONFLICT', 'user is already a member', 409)
    await expect(api.addMember(TOKEN, 'g1', { user_id: 'user-42' })).rejects.toThrow('user is already a member')
  })
})

describe('removeMember', () => {
  it('calls correct endpoint', async () => {
    mockFetch(null)
    await api.removeMember(TOKEN, 'g1', 'user-42')
    const fetchMock = vi.mocked(fetch)
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/v1/iam/groups/g1/members/user-42',
      expect.objectContaining({ method: 'DELETE' }),
    )
  })
})

// ── permissions ───────────────────────────────────────────────────────────────

describe('listGroupPermissions', () => {
  it('returns permissions response', async () => {
    mockFetch({ permissions: ['pms.product.create'] })
    const result = await api.listGroupPermissions(TOKEN, 'g1')
    expect(result.permissions).toContain('pms.product.create')
  })
})

describe('assignPermission', () => {
  it('calls correct endpoint', async () => {
    mockFetch(null)
    await api.assignPermission(TOKEN, 'g1', { rule: 'pms.product.create' })
    const fetchMock = vi.mocked(fetch)
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/v1/iam/groups/g1/permissions',
      expect.objectContaining({ method: 'POST' }),
    )
  })
})

describe('revokePermission', () => {
  it('calls correct endpoint with rule in path', async () => {
    mockFetch(null)
    await api.revokePermission(TOKEN, 'g1', 'pms.product.create')
    const fetchMock = vi.mocked(fetch)
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/v1/iam/groups/g1/permissions/pms.product.create',
      expect.objectContaining({ method: 'DELETE' }),
    )
  })
})

describe('getUserEffectivePermissions', () => {
  it('returns effective permissions', async () => {
    mockFetch({ permissions: ['pms.product.create', 'oms.order.read'] })
    const result = await api.getUserEffectivePermissions(TOKEN, 'user-1')
    expect(result.permissions).toHaveLength(2)
  })
})
