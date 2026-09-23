// @vitest-environment jsdom
import { render, screen, waitFor, fireEvent, act } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, useLocation } from 'react-router-dom'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import CommandPalette from '../CommandPalette'
import { AppProvider, useApp } from '../../../context/AppContext'

// Exposes the router's current pathname as text so tests can assert on
// where a click navigated to, without needing a full <Routes> tree.
function LocationDisplay() {
  return <span data-testid="location">{useLocation().pathname}</span>
}

vi.mock('../../../auth/AuthContext', () => ({
  useAuth: () => ({ token: 'test-token', user: null, login: vi.fn(), logout: vi.fn(), isAuthenticated: true }),
}))

vi.mock('../../../search/api', () => ({
  globalSearch: vi.fn(),
}))

import * as searchApi from '../../../search/api'

beforeEach(() => { vi.clearAllMocks() })

// CommandPalette has no trigger of its own -- OpenerButton mirrors Topbar's
// real "Search everything..." button (setCommandOpen(true)) without
// pulling in Topbar's own unrelated dependencies (scanner, notifications).
function OpenerButton() {
  const { setCommandOpen } = useApp()
  return <button onClick={() => setCommandOpen(true)}>open palette</button>
}

function renderPalette() {
  return render(
    <MemoryRouter>
      <AppProvider>
        <OpenerButton />
        <LocationDisplay />
        <CommandPalette />
      </AppProvider>
    </MemoryRouter>,
  )
}

async function openPalette() {
  await userEvent.click(screen.getByText('open palette'))
}

describe('CommandPalette', () => {
  it('is closed by default and opens on click', async () => {
    renderPalette()
    expect(screen.queryByPlaceholderText('Search everything...')).not.toBeInTheDocument()

    await openPalette()

    expect(screen.getByPlaceholderText('Search everything...')).toBeInTheDocument()
  })

  it('opens on Cmd+K as a search dialog and closes on Escape', async () => {
    renderPalette()

    act(() => { fireEvent.keyDown(window, { key: 'k', metaKey: true }) })
    expect(screen.getByRole('dialog', { name: 'Search' })).toBeInTheDocument()
    expect(screen.getByPlaceholderText('Search everything...')).toHaveFocus()

    await userEvent.keyboard('{Escape}')
    expect(screen.queryByPlaceholderText('Search everything...')).not.toBeInTheDocument()
  })

  it('shows navigation commands filtered instantly, without calling the search API', async () => {
    renderPalette()
    await openPalette()

    await userEvent.type(screen.getByPlaceholderText('Search everything...'), 'suppl')

    expect(screen.getByText('Go to Suppliers')).toBeInTheDocument()
    expect(screen.queryByText('Go to Products')).not.toBeInTheDocument()
    expect(searchApi.globalSearch).not.toHaveBeenCalled()
  })

  it('debounces and calls the backend search once the query is 2+ characters, grouped by module', async () => {
    vi.mocked(searchApi.globalSearch).mockResolvedValue({
      products: [{ id: 'prod-1', name: 'Açaí Bowl', slug: 'acai-bowl' }],
      suppliers: [{ id: 'sup-1', name: 'Açaí Distribuidora', email: 'contact@acai.example' }],
      customers: [],
      sales_orders: [{ id: 'so-1', number: 'SO-10042', billing_first_name: 'Jane', billing_last_name: 'Doe', status: 'completed' }],
    })
    renderPalette()
    await openPalette()

    await userEvent.type(screen.getByPlaceholderText('Search everything...'), 'acai')

    await waitFor(() => expect(searchApi.globalSearch).toHaveBeenCalledWith('test-token', 'acai'))
    expect(await screen.findByText('Açaí Bowl')).toBeInTheDocument()
    expect(screen.getByText('Açaí Distribuidora')).toBeInTheDocument()
    expect(screen.getByText('#SO-10042')).toBeInTheDocument()
    expect(screen.getByText('Products')).toBeInTheDocument()
    expect(screen.getByText('Suppliers')).toBeInTheDocument()
    expect(screen.getByText('Sales Orders')).toBeInTheDocument()
    // Customers matched nothing -- its section header must not render.
    expect(screen.queryByText('Customers')).not.toBeInTheDocument()
  })

  it('does not call the backend for a single-character query', async () => {
    renderPalette()
    await openPalette()

    await userEvent.type(screen.getByPlaceholderText('Search everything...'), 'a')
    await new Promise(r => setTimeout(r, 300))

    expect(searchApi.globalSearch).not.toHaveBeenCalled()
  })

  it('clicking a product result navigates to its edit page (a real per-record URL, not a query param) and closes the palette', async () => {
    vi.mocked(searchApi.globalSearch).mockResolvedValue({
      products: [{ id: 'prod-1', name: 'Açaí Bowl', slug: 'acai-bowl' }],
      suppliers: [], customers: [], sales_orders: [],
    })
    renderPalette()
    await openPalette()
    await userEvent.type(screen.getByPlaceholderText('Search everything...'), 'acai')
    await screen.findByText('Açaí Bowl')

    await userEvent.click(screen.getByText('Açaí Bowl'))

    expect(screen.queryByPlaceholderText('Search everything...')).not.toBeInTheDocument()
    expect(screen.getByTestId('location')).toHaveTextContent('/products/prod-1')
  })

  it('clicking a sales-order result navigates to its own URL, same as the other entities', async () => {
    vi.mocked(searchApi.globalSearch).mockResolvedValue({
      products: [], suppliers: [], customers: [],
      sales_orders: [{ id: 'so-42', number: '22455', billing_first_name: '', billing_last_name: '', status: 'completed' }],
    })
    renderPalette()
    await openPalette()
    await userEvent.type(screen.getByPlaceholderText('Search everything...'), '22455')
    await screen.findByText('#22455')

    await userEvent.click(screen.getByText('#22455'))

    expect(screen.getByTestId('location')).toHaveTextContent('/sales-orders/so-42')
  })

  it('shows a search-failed message without crashing', async () => {
    vi.mocked(searchApi.globalSearch).mockRejectedValue(new Error('search backend down'))
    renderPalette()
    await openPalette()

    await userEvent.type(screen.getByPlaceholderText('Search everything...'), 'acai')

    expect(await screen.findByText('search backend down')).toBeInTheDocument()
  })

  it('offers the sidebar pages under the same names', async () => {
    renderPalette()
    await openPalette()

    expect(screen.getByText('Go to Sales Orders')).toBeInTheDocument()
    expect(screen.getByText('Go to Reports: Overview')).toBeInTheDocument()
  })

  it('shows the real g-then-letter shortcut next to the pages that have one', async () => {
    renderPalette()
    await openPalette()

    const products = screen.getByText('Go to Products').closest('button')!
    expect(products).toHaveTextContent('G P')
    const warehouses = screen.getByText('Go to Warehouses').closest('button')!
    expect(warehouses).not.toHaveTextContent(/G [A-Z]$/)
  })

  it('is a combobox whose active result screen readers can follow', async () => {
    renderPalette()
    await openPalette()

    const input = screen.getByRole('combobox', { name: 'Search' })
    const listbox = screen.getByRole('listbox')
    expect(input).toHaveAttribute('aria-controls', listbox.id)

    const first = screen.getAllByRole('option')[0]
    expect(first).toHaveAttribute('aria-selected', 'true')
    expect(input).toHaveAttribute('aria-activedescendant', first.id)

    await userEvent.keyboard('{ArrowDown}')

    const second = screen.getAllByRole('option')[1]
    expect(second).toHaveAttribute('aria-selected', 'true')
    expect(first).toHaveAttribute('aria-selected', 'false')
    expect(input).toHaveAttribute('aria-activedescendant', second.id)
  })

  it('groups results under their section names', async () => {
    renderPalette()
    await openPalette()
    expect(screen.getByRole('group', { name: 'Navigation' })).toBeInTheDocument()
  })

  it('highlights the row under the pointer, not the last row', async () => {
    renderPalette()
    await openPalette()

    const options = screen.getAllByRole('option')
    await userEvent.hover(options[1])

    expect(options[1]).toHaveAttribute('aria-selected', 'true')
    expect(options[options.length - 1]).toHaveAttribute('aria-selected', 'false')
  })
})
