// @vitest-environment jsdom
import { render, screen, waitFor, act } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import Sidebar from '../Sidebar'
import { AppProvider } from '../../../context/AppContext'
import type { Product } from '../../../pms/types'

vi.mock('../../../auth/AuthContext', () => ({
  useAuth: () => ({ token: 'test-token', user: null, login: vi.fn(), logout: vi.fn(), isAuthenticated: true }),
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

function renderSidebar() {
  return render(<MemoryRouter><AppProvider><Sidebar /></AppProvider></MemoryRouter>)
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

describe('Sidebar Intelligence section', () => {
  beforeEach(() => vi.mocked(listProducts).mockResolvedValue({ products: [], total: 0 }))

  it('groups Promotions and Reports under an Intelligence section header', () => {
    renderSidebar()

    const section = screen.getByText('Intelligence')
    const promotions = screen.getByRole('button', { name: /Promotions/ })
    const reports = screen.getByRole('button', { name: /Reports/ })
    const settings = screen.getByRole('button', { name: /Settings/ })

    // Reports is now a collapsible submenu (its button sits inside its own
    // wrapper div, not as a direct sibling of the other nav buttons), so
    // sibling-index comparison no longer reflects nav order -- document
    // order does, regardless of nesting depth.
    const isBefore = (a: Element, b: Element) => !!(a.compareDocumentPosition(b) & Node.DOCUMENT_POSITION_FOLLOWING)
    expect(isBefore(section, promotions)).toBe(true)
    expect(isBefore(promotions, reports)).toBe(true)
    expect(isBefore(reports, settings)).toBe(true)
  })
})
