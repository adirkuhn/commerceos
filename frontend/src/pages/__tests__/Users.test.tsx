// @vitest-environment jsdom
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import Users from '../Users'
import type { CompanyMember, CompanyInvite } from '../../team/types'
import type { Group } from '../../iam/types'

let mockRole = 'customer_owner'
vi.mock('../../auth/AuthContext', () => ({
  useAuth: () => ({ token: 'test-token', user: { role: mockRole }, login: vi.fn(), logout: vi.fn(), isAuthenticated: true }),
}))

vi.mock('../../team/api', () => ({
  listMembers: vi.fn(),
  updateMemberRole: vi.fn(),
  removeMember: vi.fn(),
  listInvites: vi.fn(),
  createInvite: vi.fn(),
  revokeInvite: vi.fn(),
}))

vi.mock('../../iam/api', () => ({
  listGroups: vi.fn(),
  createGroup: vi.fn(),
  updateGroup: vi.fn(),
  deleteGroup: vi.fn(),
  listMembers: vi.fn(),
  addMember: vi.fn(),
  removeMember: vi.fn(),
  listGroupPermissions: vi.fn(),
  assignPermission: vi.fn(),
  revokePermission: vi.fn(),
}))

import * as teamApi from '../../team/api'
import * as iamApi from '../../iam/api'

function member(overrides: Partial<CompanyMember> = {}): CompanyMember {
  return {
    membership_id: 'm-1', user_id: 'u-1', email: 'alex@example.com', first_name: 'Alex', last_name: 'Kuhn',
    role: 'customer_owner', status: 'active', last_login_at: '2026-08-20T00:00:00Z',
    ...overrides,
  }
}

function invite(overrides: Partial<CompanyInvite> = {}): CompanyInvite {
  return {
    id: 'inv-1', email: 'new@example.com', role: 'customer_user',
    expires_at: '2026-08-29T00:00:00Z', created_at: '2026-08-22T00:00:00Z',
    ...overrides,
  }
}

function group(overrides: Partial<Group> = {}): Group {
  return {
    id: 'g-1', tenant_id: 't-1', name: 'Warehouse Staff', description: 'Pickers and packers',
    created_at: '2026-08-01T00:00:00Z', updated_at: '2026-08-01T00:00:00Z',
    ...overrides,
  }
}

beforeEach(() => {
  vi.clearAllMocks()
  mockRole = 'customer_owner'
  vi.mocked(teamApi.listMembers).mockResolvedValue([])
  vi.mocked(teamApi.listInvites).mockResolvedValue([])
  vi.mocked(iamApi.listGroups).mockResolvedValue([])
})

describe('Users page — real member data', () => {
  it('renders real members from lm2, not mock data', async () => {
    vi.mocked(teamApi.listMembers).mockResolvedValue([member()])
    render(<Users />)
    expect(await screen.findByText('Alex Kuhn')).toBeInTheDocument()
    expect(screen.getByText('alex@example.com')).toBeInTheDocument()
    expect(teamApi.listMembers).toHaveBeenCalledWith('test-token')
  })

  it('changes a member role', async () => {
    vi.mocked(teamApi.listMembers).mockResolvedValue([member()])
    vi.mocked(teamApi.updateMemberRole).mockResolvedValue(null)
    render(<Users />)
    await screen.findByText('Alex Kuhn')

    fireEvent.change(screen.getByLabelText(/role for alex kuhn/i), { target: { value: 'customer_user' } })

    await waitFor(() => expect(teamApi.updateMemberRole).toHaveBeenCalledWith('test-token', 'm-1', 'customer_user'))
  })

  it('removes a member after confirming', async () => {
    vi.mocked(teamApi.listMembers).mockResolvedValue([member()])
    vi.mocked(teamApi.removeMember).mockResolvedValue(null)
    vi.spyOn(window, 'confirm').mockReturnValue(true)
    render(<Users />)
    await screen.findByText('Alex Kuhn')

    fireEvent.click(screen.getByLabelText(/remove alex kuhn/i))

    await waitFor(() => expect(teamApi.removeMember).toHaveBeenCalledWith('test-token', 'm-1'))
  })

  it('surfaces an error removing the last owner', async () => {
    vi.mocked(teamApi.listMembers).mockResolvedValue([member()])
    vi.mocked(teamApi.removeMember).mockRejectedValue(new Error('company must have at least one owner'))
    vi.spyOn(window, 'confirm').mockReturnValue(true)
    render(<Users />)
    await screen.findByText('Alex Kuhn')

    fireEvent.click(screen.getByLabelText(/remove alex kuhn/i))

    expect(await screen.findByText('company must have at least one owner')).toBeInTheDocument()
  })
})

describe('Users page — invites', () => {
  it('sends an invite', async () => {
    vi.mocked(teamApi.createInvite).mockResolvedValue({ id: 'inv-2', email: 'joe@example.com', role: 'customer_user' })
    render(<Users />)
    await waitFor(() => expect(teamApi.listMembers).toHaveBeenCalled())

    fireEvent.click(screen.getByRole('button', { name: /invite user/i }))
    fireEvent.change(screen.getByLabelText(/email/i), { target: { value: 'joe@example.com' } })
    fireEvent.click(screen.getByRole('button', { name: /send invite/i }))

    await waitFor(() => expect(teamApi.createInvite).toHaveBeenCalledWith('test-token', 'joe@example.com', 'customer_user'))
  })

  it('lists pending invites and revokes one', async () => {
    vi.mocked(teamApi.listInvites).mockResolvedValue([invite()])
    vi.mocked(teamApi.revokeInvite).mockResolvedValue(null)
    render(<Users />)
    expect(await screen.findByText('new@example.com')).toBeInTheDocument()

    fireEvent.click(screen.getByLabelText(/revoke invite for new@example.com/i))

    await waitFor(() => expect(teamApi.revokeInvite).toHaveBeenCalledWith('test-token', 'inv-1'))
  })
})

describe('Users page — groups panel', () => {
  it('shows real groups and adds a real member to a group', async () => {
    vi.mocked(teamApi.listMembers).mockResolvedValue([member()])
    vi.mocked(iamApi.listGroups).mockResolvedValue([group()])
    vi.mocked(iamApi.listMembers).mockResolvedValue({ members: [] })
    vi.mocked(iamApi.addMember).mockResolvedValue(null)
    render(<Users />)
    await screen.findByText('Alex Kuhn')

    fireEvent.click(screen.getByRole('button', { name: /groups/i }))
    fireEvent.click(await screen.findByText('Warehouse Staff'))
    fireEvent.change(await screen.findByLabelText(/add member to group/i), { target: { value: 'u-1' } })
    fireEvent.click(screen.getByRole('button', { name: /^add$/i }))

    await waitFor(() => expect(iamApi.addMember).toHaveBeenCalledWith('test-token', 'g-1', { user_id: 'u-1' }))
  })
})

describe('Users page — access control', () => {
  it('shows a not-authorized message for a non-owner', async () => {
    mockRole = 'customer_user'
    render(<Users />)
    expect(await screen.findByText(/only.*owner/i)).toBeInTheDocument()
    expect(teamApi.listMembers).not.toHaveBeenCalled()
  })

  it('allows super_admin through, same as customer_owner', async () => {
    mockRole = 'super_admin'
    render(<Users />)
    await waitFor(() => expect(teamApi.listMembers).toHaveBeenCalledWith('test-token'))
    expect(screen.queryByText(/only.*owner/i)).not.toBeInTheDocument()
  })
})
