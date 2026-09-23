// @vitest-environment jsdom
import { screen, fireEvent, waitFor } from '@testing-library/react'
import { act } from 'react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import Users from '../Users'
import { renderAt } from '../../test/renderAt'
import type { CompanyMember, CompanyInvite } from '../../team/types'
import type { Group } from '../../iam/types'

let mockRole = 'customer_owner'
let mockUserId = 'logged-in-user'
vi.mock('../../auth/AuthContext', () => ({
  useAuth: () => ({ token: 'test-token', user: { role: mockRole, user_id: mockUserId }, login: vi.fn(), logout: vi.fn(), isAuthenticated: true }),
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
  getUserActivities: vi.fn(),
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
  mockUserId = 'logged-in-user'
  vi.mocked(teamApi.listMembers).mockResolvedValue([])
  vi.mocked(teamApi.listInvites).mockResolvedValue([])
  vi.mocked(iamApi.listGroups).mockResolvedValue([])
  vi.mocked(iamApi.getUserActivities).mockResolvedValue({ activities: [] })
})

describe('Users page — real member data', () => {
  it('renders real members from lm2, not mock data', async () => {
    vi.mocked(teamApi.listMembers).mockResolvedValue([member()])
    renderAt(<Users />, '/users')
    expect(await screen.findByText('Alex Kuhn')).toBeInTheDocument()
    expect(screen.getByText('alex@example.com')).toBeInTheDocument()
    expect(teamApi.listMembers).toHaveBeenCalledWith('test-token')
  })

  it('clicking a member opens their activity panel', async () => {
    vi.mocked(teamApi.listMembers).mockResolvedValue([member()])
    vi.mocked(iamApi.getUserActivities).mockResolvedValue({
      activities: [{
        id: 'a-1', sales_order_id: 'so-1', user_id: 'u-1', activity_type: 'myparcel_shipment_created',
        summary: 'Shipment created via MyParcel', metadata: {}, created_at: '2026-08-26T00:00:00Z',
      }],
    })
    renderAt(<Users />, '/users')
    await screen.findByText('Alex Kuhn')

    fireEvent.click(screen.getByTitle("View Alex Kuhn's activity"))

    expect(await screen.findByText('Shipment created via MyParcel')).toBeInTheDocument()
    expect(iamApi.getUserActivities).toHaveBeenCalledWith('test-token', 'u-1')
  })

  it('changes a member role', async () => {
    vi.mocked(teamApi.listMembers).mockResolvedValue([member()])
    vi.mocked(teamApi.updateMemberRole).mockResolvedValue(null)
    renderAt(<Users />, '/users')
    await screen.findByText('Alex Kuhn')

    fireEvent.change(screen.getByLabelText(/role for alex kuhn/i), { target: { value: 'customer_user' } })

    await waitFor(() => expect(teamApi.updateMemberRole).toHaveBeenCalledWith('test-token', 'm-1', 'customer_user'))
  })

  it('removes a member after confirming', async () => {
    vi.mocked(teamApi.listMembers).mockResolvedValue([member()])
    vi.mocked(teamApi.removeMember).mockResolvedValue(null)
    vi.spyOn(window, 'confirm').mockReturnValue(true)
    renderAt(<Users />, '/users')
    await screen.findByText('Alex Kuhn')

    fireEvent.click(screen.getByLabelText(/remove alex kuhn/i))

    await waitFor(() => expect(teamApi.removeMember).toHaveBeenCalledWith('test-token', 'm-1'))
  })

  it('hides the Remove control on the logged-in user\'s own row', async () => {
    mockUserId = 'u-1'
    vi.mocked(teamApi.listMembers).mockResolvedValue([member()])
    renderAt(<Users />, '/users')
    await screen.findByText('Alex Kuhn')

    expect(screen.queryByLabelText(/remove alex kuhn/i)).not.toBeInTheDocument()
  })

  it('surfaces an error removing the last owner', async () => {
    vi.mocked(teamApi.listMembers).mockResolvedValue([member()])
    vi.mocked(teamApi.removeMember).mockRejectedValue(new Error('company must have at least one owner'))
    vi.spyOn(window, 'confirm').mockReturnValue(true)
    renderAt(<Users />, '/users')
    await screen.findByText('Alex Kuhn')

    fireEvent.click(screen.getByLabelText(/remove alex kuhn/i))

    expect(await screen.findByText('company must have at least one owner')).toBeInTheDocument()
  })
})

describe('Users page — invites', () => {
  it('sends an invite', async () => {
    vi.mocked(teamApi.createInvite).mockResolvedValue({ id: 'inv-2', email: 'joe@example.com', role: 'customer_user' })
    renderAt(<Users />, '/users')
    await waitFor(() => expect(teamApi.listMembers).toHaveBeenCalled())

    fireEvent.click(screen.getByRole('button', { name: /invite user/i }))
    fireEvent.change(screen.getByLabelText(/email/i), { target: { value: 'joe@example.com' } })
    fireEvent.click(screen.getByRole('button', { name: /send invite/i }))

    await waitFor(() => expect(teamApi.createInvite).toHaveBeenCalledWith('test-token', 'joe@example.com', 'customer_user'))
  })

  it('lists pending invites and revokes one', async () => {
    vi.mocked(teamApi.listInvites).mockResolvedValue([invite()])
    vi.mocked(teamApi.revokeInvite).mockResolvedValue(null)
    renderAt(<Users />, '/users')
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
    renderAt(<Users />, '/users')
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
    renderAt(<Users />, '/users')
    expect(await screen.findByText(/only.*owner/i)).toBeInTheDocument()
    expect(teamApi.listMembers).not.toHaveBeenCalled()
  })

  it('allows super_admin through, same as customer_owner', async () => {
    mockRole = 'super_admin'
    renderAt(<Users />, '/users')
    await waitFor(() => expect(teamApi.listMembers).toHaveBeenCalledWith('test-token'))
    expect(screen.queryByText(/only.*owner/i)).not.toBeInTheDocument()
  })
})

describe('Users URLs (refresh + Back)', () => {
  it('clicking a member pushes /users/:userId; that URL reopens the panel on load', async () => {
    vi.mocked(teamApi.listMembers).mockResolvedValue([member()])
    const { router, unmount } = renderAt(<Users />, '/users')
    fireEvent.click(await screen.findByTitle("View Alex Kuhn's activity"))
    expect(router.state.location.pathname).toBe('/users/u-1')
    unmount()

    renderAt(<Users />, '/users', '/users/u-1')
    await waitFor(() => expect(iamApi.getUserActivities).toHaveBeenCalledWith('test-token', 'u-1'))
  })

  it('Back closes the member panel', async () => {
    vi.mocked(teamApi.listMembers).mockResolvedValue([member()])
    const { router } = renderAt(<Users />, '/users')
    fireEvent.click(await screen.findByTitle("View Alex Kuhn's activity"))
    await waitFor(() => expect(iamApi.getUserActivities).toHaveBeenCalled())

    await act(() => router.navigate(-1))
    expect(router.state.location.pathname).toBe('/users')
  })

  it('/users/groups/:groupId opens the groups panel with that group selected', async () => {
    vi.mocked(iamApi.listGroups).mockResolvedValue([group()])
    vi.mocked(iamApi.listMembers).mockResolvedValue({ members: [] })
    renderAt(<Users />, '/users', '/users/groups/g-1')
    expect(await screen.findByLabelText(/add member to group/i)).toBeInTheDocument()
  })

  it('selecting a group pushes /users/groups/:groupId', async () => {
    vi.mocked(iamApi.listGroups).mockResolvedValue([group()])
    vi.mocked(iamApi.listMembers).mockResolvedValue({ members: [] })
    const { router } = renderAt(<Users />, '/users', '/users/groups')
    fireEvent.click(await screen.findByText('Warehouse Staff'))
    expect(router.state.location.pathname).toBe('/users/groups/g-1')
  })

  it('/users/invite opens the invite form on load', async () => {
    renderAt(<Users />, '/users', '/users/invite')
    expect(await screen.findByRole('button', { name: /send invite/i })).toBeInTheDocument()
  })

  it('search lives in ?q=', async () => {
    vi.mocked(teamApi.listMembers).mockResolvedValue([member(), member({ membership_id: 'm-2', user_id: 'u-2', first_name: 'Bo', last_name: 'Li', email: 'bo@example.com' })])
    renderAt(<Users />, '/users', '/users?q=bo')
    expect(await screen.findByText('Bo Li')).toBeInTheDocument()
    expect(screen.queryByText('Alex Kuhn')).not.toBeInTheDocument()
  })
})
