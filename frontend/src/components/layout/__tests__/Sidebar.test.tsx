// @vitest-environment jsdom
import { render, screen, waitFor } from '@testing-library/react'
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

function product(id: string): Product {
  return {
    id, tenant_id: 't1', name: `Product ${id}`, slug: `product-${id}`,
    description: '', status: 'active', has_variants: false, created_at: '', updated_at: '',
  }
}

function renderSidebar() {
  return render(<AppProvider><Sidebar /></AppProvider>)
}

describe('Sidebar product count badge', () => {
  beforeEach(() => vi.mocked(listProducts).mockReset())

  it('shows the real product count, not a hardcoded value', async () => {
    vi.mocked(listProducts).mockResolvedValue([product('1'), product('2'), product('3')])
    renderSidebar()
    await waitFor(() => expect(screen.getByText('3')).toBeInTheDocument())
    expect(screen.queryByText('2.4k')).not.toBeInTheDocument()
  })

  it('hides the badge when there are no products', async () => {
    vi.mocked(listProducts).mockResolvedValue([])
    renderSidebar()
    await waitFor(() => expect(listProducts).toHaveBeenCalled())
    expect(screen.queryByText('0')).not.toBeInTheDocument()
  })
})
