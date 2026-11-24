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

    const orders = screen.getByRole('button', { name: 'Orders' })
    expect(orders).toHaveAttribute('aria-expanded', 'true')
    expect(screen.getByRole('button', { name: 'Sales Orders' })).toBeInTheDocument()

    await userEvent.click(orders)

    expect(orders).toHaveAttribute('aria-expanded', 'false')
    expect(screen.queryByRole('button', { name: 'Sales Orders' })).not.toBeInTheDocument()
  })

  it('marks the current page', () => {
    renderSidebar('/customers')
    expect(screen.getByRole('button', { name: 'Customers' })).toHaveAttribute('aria-current', 'page')
    expect(screen.getByRole('button', { name: 'Suppliers' })).not.toHaveAttribute('aria-current')
  })

  it('shows no made-up status badges', async () => {
    renderSidebar()
    await userEvent.click(screen.getByRole('button', { name: 'Setup' }))
    expect(screen.getByRole('button', { name: 'Integrations' })).toBeInTheDocument()
  })

  it('hides team management from non-owners', async () => {
    auth.role = 'customer_user'
    renderSidebar()
    await userEvent.click(screen.getByRole('button', { name: 'Setup' }))
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

describe('Sidebar look switch', () => {
  beforeEach(() => {
    vi.mocked(listProducts).mockResolvedValue({ products: [], total: 0 })
    auth.role = 'customer_owner'
    localStorage.clear()
    document.documentElement.classList.remove('look-wallet', 'look-graphite')
  })

  it('switches between the Ticket Wallet and Graphite looks from the account menu', async () => {
    renderSidebar()

    await userEvent.click(screen.getByRole('button', { name: 'Account: Owner' }))
    expect(screen.getByRole('menuitemradio', { name: 'Ticket Wallet' })).toHaveAttribute('aria-checked', 'true')
    await userEvent.click(screen.getByRole('menuitemradio', { name: 'Graphite' }))

    expect(document.documentElement.classList.contains('look-graphite')).toBe(true)
    expect(localStorage.getItem('commerceos.look')).toBe('graphite')

    await userEvent.click(screen.getByRole('button', { name: 'Account: Owner' }))
    expect(screen.getByRole('menuitemradio', { name: 'Graphite' })).toHaveAttribute('aria-checked', 'true')
    await userEvent.click(screen.getByRole('menuitemradio', { name: 'Ticket Wallet' }))

    expect(document.documentElement.classList.contains('look-wallet')).toBe(true)
    expect(document.documentElement.classList.contains('look-graphite')).toBe(false)
  })
})

describe('Sidebar navigation', () => {
  beforeEach(() => {
    vi.mocked(listProducts).mockResolvedValue({ products: [], total: 0 })
    auth.role = 'customer_owner'
  })

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

describe('Sidebar deployed version', () => {
  beforeEach(() => vi.mocked(listProducts).mockResolvedValue({ products: [], total: 0 }))
  afterEach(() => {
    delete globalThis.__COMMERCEOS_CONFIG__
    auth.role = 'customer_owner'
  })

  it('shows the deployed version under the role', async () => {
    auth.role = 'super_admin'
    globalThis.__COMMERCEOS_CONFIG__ = { COMMERCEOS_VERSION: '1.0.11' }

    renderSidebar()

    const account = await screen.findByRole('button', { name: /Account: Platform admin/ })
    expect(within(account).getByText('Platform admin')).toBeInTheDocument()
    expect(within(account).getByText('v1.0.11')).toBeInTheDocument()
  })

  it('keeps a leading v the server already wrote', async () => {
    globalThis.__COMMERCEOS_CONFIG__ = { COMMERCEOS_VERSION: 'v1.0.11' }

    renderSidebar()

    expect(await screen.findByText('v1.0.11')).toBeInTheDocument()
  })

  it('shows no version when none is deployed (local dev)', async () => {
    renderSidebar()

    await screen.findByRole('button', { name: /Account: Owner/ })
    expect(screen.queryByText(/^v\d/)).not.toBeInTheDocument()
  })
})
