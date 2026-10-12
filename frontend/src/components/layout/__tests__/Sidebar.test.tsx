// @vitest-environment jsdom
import { render, screen, waitFor, act, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
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

describe('Sidebar brand', () => {
  beforeEach(() => vi.mocked(listProducts).mockResolvedValue({ products: [], total: 0 }))

  it('shows the product name beside a decorative mark', () => {
    const { container } = renderSidebar()
    expect(screen.getByText('CommerceOS')).toBeInTheDocument()
    expect(container.querySelector('aside svg[aria-hidden="true"]')).not.toBeNull()
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

describe('Sidebar design switch', () => {
  beforeEach(() => {
    vi.mocked(listProducts).mockResolvedValue({ products: [], total: 0 })
    auth.role = 'customer_owner'
    localStorage.clear()
    document.documentElement.classList.remove('design-depot')
  })

  it('switches between the depot and classic designs from the account menu', async () => {
    document.documentElement.classList.add('design-depot')
    renderSidebar()
    expect(screen.getByRole('button', { name: 'Today' })).toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Account: Owner' }))
    await userEvent.click(screen.getByRole('menuitem', { name: 'Use the classic design' }))

    expect(document.documentElement.classList.contains('design-depot')).toBe(false)
    expect(localStorage.getItem('commerceos.design')).toBe('classic')
    expect(screen.getByRole('button', { name: 'Dashboard' })).toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Account: Owner' }))
    await userEvent.click(screen.getByRole('menuitem', { name: 'Use the new design' }))

    expect(document.documentElement.classList.contains('design-depot')).toBe(true)
  })
})

describe('Sidebar depot navigation', () => {
  beforeEach(() => {
    vi.mocked(listProducts).mockResolvedValue({ products: [], total: 0 })
    auth.role = 'customer_owner'
    document.documentElement.classList.add('design-depot')
  })
  afterEach(() => document.documentElement.classList.remove('design-depot'))

  it('shows one Reports link, marked current on any report page', () => {
    renderSidebar('/reports/customer-rfm')
    const reports = screen.getByRole('button', { name: 'Reports' })
    expect(reports).toHaveAttribute('aria-current', 'page')
    expect(screen.queryByRole('button', { name: 'Customer Segments (RFM)' })).not.toBeInTheDocument()
  })

  it('keeps Setup closed until opened, and open when a setup page is current', async () => {
    renderSidebar('/')
    expect(screen.queryByRole('button', { name: 'Webhooks' })).not.toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Setup' }))
    expect(screen.getByRole('button', { name: 'Webhooks' })).toBeInTheDocument()
  })

  it('pins Setup below the scrolling list, so it is reachable on short screens', () => {
    renderSidebar('/')
    const pinned = screen.getByTestId('nav-pinned')
    expect(within(pinned).getByRole('button', { name: 'Setup' })).toBeInTheDocument()
    expect(within(screen.getByTestId('nav-scroll')).queryByRole('button', { name: 'Setup' })).not.toBeInTheDocument()
  })

  it('opens Setup when a setup page is the current page', () => {
    renderSidebar('/webhooks')
    expect(screen.getByRole('button', { name: 'Webhooks' })).toHaveAttribute('aria-current', 'page')
  })
})
