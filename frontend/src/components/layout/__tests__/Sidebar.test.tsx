// @vitest-environment jsdom
import { render, screen, waitFor, act, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import Sidebar from '../Sidebar'
import { AppProvider } from '../../../context/AppContext'
import type { Product } from '../../../pms/types'

const auth = vi.hoisted(() => ({ logout: vi.fn(), role: 'customer_owner' }))
vi.mock('../../../auth/AuthContext', () => ({
  useAuth: () => ({
    token: 'test-token', user: { user_id: 'u1', tenant_id: 't1', role: auth.role, exp: 0, iat: 0 },
    login: vi.fn(), logout: auth.logout, isAuthenticated: true,
  }),
}))

vi.mock('../../../pms/api', () => ({
  listProducts: vi.fn(),
}))

import { listProducts } from '../../../pms/api'
import { notifyProductsChanged } from '../../../pms/productsBus'

function product(id: string): Product {
  return {
    id, tenant_id: 't1', name: `Product ${id}`, slug: `product-${id}`,
    description: '', status: 'active', has_variants: false, tags: [], created_at: '', updated_at: '',
  }
}

function renderSidebar(url = '/') {
  return render(<MemoryRouter initialEntries={[url]}><AppProvider><Sidebar /></AppProvider></MemoryRouter>)
}

describe('Sidebar product count badge', () => {
  beforeEach(() => vi.mocked(listProducts).mockReset())

  it('shows the real product count, not a hardcoded value', async () => {
    vi.mocked(listProducts).mockResolvedValue({ products: [product('1')], total: 3 })
    renderSidebar()
    await waitFor(() => expect(screen.getByText('3')).toBeInTheDocument())
    expect(screen.queryByText('2.4k')).not.toBeInTheDocument()
  })

  it('hides the badge when there are no products', async () => {
    vi.mocked(listProducts).mockResolvedValue({ products: [], total: 0 })
    renderSidebar()
    await waitFor(() => expect(listProducts).toHaveBeenCalled())
    expect(screen.queryByText('0')).not.toBeInTheDocument()
  })

  it('refetches when notified of a products change elsewhere (e.g. bulk import/delete)', async () => {
    vi.mocked(listProducts).mockResolvedValue({ products: [product('1')], total: 3 })
    renderSidebar()
    await waitFor(() => expect(screen.getByText('3')).toBeInTheDocument())

    vi.mocked(listProducts).mockResolvedValue({ products: [], total: 0 })
    act(() => { notifyProductsChanged() })

    await waitFor(() => expect(screen.queryByText('3')).not.toBeInTheDocument())
  })
})

describe('Sidebar groups', () => {
  beforeEach(() => {
    vi.mocked(listProducts).mockResolvedValue({ products: [], total: 0 })
    auth.role = 'customer_owner'
  })

  it('groups pages under disclosure headers', async () => {
    renderSidebar()

    const sell = screen.getByRole('button', { name: 'Sell' })
    expect(sell).toHaveAttribute('aria-expanded', 'true')
    expect(screen.getByRole('button', { name: 'Sales Orders' })).toBeInTheDocument()

    await userEvent.click(sell)

    expect(sell).toHaveAttribute('aria-expanded', 'false')
    expect(screen.queryByRole('button', { name: 'Sales Orders' })).not.toBeInTheDocument()
  })

  it('keeps Reports closed until opened, and open when a report is the current page', async () => {
    const { unmount } = renderSidebar()
    expect(screen.getByRole('button', { name: 'Reports' })).toHaveAttribute('aria-expanded', 'false')
    expect(screen.queryByRole('button', { name: 'Sales Trends' })).not.toBeInTheDocument()
    unmount()

    renderSidebar('/reports/sales-trends')
    expect(screen.getByRole('button', { name: 'Reports' })).toHaveAttribute('aria-expanded', 'true')
    expect(screen.getByRole('button', { name: 'Sales Trends' })).toHaveAttribute('aria-current', 'page')
  })

  it('marks the current page', () => {
    renderSidebar('/customers')
    expect(screen.getByRole('button', { name: 'Customers' })).toHaveAttribute('aria-current', 'page')
    expect(screen.getByRole('button', { name: 'Suppliers' })).not.toHaveAttribute('aria-current')
  })

  it('shows no made-up status badges', () => {
    renderSidebar()
    expect(screen.getByRole('button', { name: 'Integrations' })).toBeInTheDocument()
  })

  it('hides team management from non-owners', () => {
    auth.role = 'customer_user'
    renderSidebar()
    expect(screen.queryByRole('button', { name: 'Users & Roles' })).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Settings' })).toBeInTheDocument()
  })
})

describe('Sidebar account menu', () => {
  beforeEach(() => {
    vi.mocked(listProducts).mockResolvedValue({ products: [], total: 0 })
    auth.role = 'customer_owner'
    auth.logout.mockReset()
  })

  it('opens a menu with the role instead of signing out on click', async () => {
    renderSidebar()

    await userEvent.click(screen.getByRole('button', { name: 'Account: Owner' }))

    expect(auth.logout).not.toHaveBeenCalled()
    const menu = screen.getByRole('menu')
    expect(within(menu).getByText('Owner')).toBeInTheDocument()
  })

  it('signs out from the menu', async () => {
    renderSidebar()
    await userEvent.click(screen.getByRole('button', { name: 'Account: Owner' }))

    await userEvent.click(screen.getByRole('menuitem', { name: 'Sign out' }))

    expect(auth.logout).toHaveBeenCalledTimes(1)
  })

  it('closes the menu on Escape', async () => {
    renderSidebar()
    await userEvent.click(screen.getByRole('button', { name: 'Account: Owner' }))

    await userEvent.keyboard('{Escape}')

    expect(screen.queryByRole('menu')).not.toBeInTheDocument()
  })
})
