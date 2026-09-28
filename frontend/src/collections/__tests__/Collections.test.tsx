// @vitest-environment jsdom
import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import Collections from '../../pages/Collections'
import { renderAt } from '../../test/renderAt'
import type { Collection } from '../types'

vi.mock('../../auth/AuthContext', () => ({ useAuth: () => ({ token: 'tok' }) }))
vi.mock('../api', () => ({
  listCollections: vi.fn(), getCollection: vi.fn(), createCollection: vi.fn(), updateCollection: vi.fn(),
  deleteCollection: vi.fn(), setCollectionProducts: vi.fn(),
}))
vi.mock('../../pms/api', () => ({ listCategories: vi.fn(), listProducts: vi.fn(), getProduct: vi.fn() }))
import * as api from '../api'
import * as pms from '../../pms/api'

const renderPage = (url = '/collections') => renderAt(<Collections />, '/collections', url)

const col = (o: Partial<Collection>): Collection => ({
  id: 'c1', name: 'Summer', slug: 'summer', description: '', seo_title: '', seo_description: '', type: 'manual', match: 'all',
  rules: [], sort: 'manual', published: true, created_at: '', updated_at: '', ...o,
})

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(api.listCollections).mockResolvedValue({ items: [col({}), col({ id: 'c2', name: 'Sale', slug: 'sale', type: 'smart', rules: [{ field: 'on_sale' }], published: false })], total: 2 })
  vi.mocked(pms.listCategories).mockResolvedValue([{ id: 'cat-1', tenant_id: 't', name: 'Shoes', slug: 'shoes', position: 0, created_at: '', updated_at: '' }])
  vi.mocked(pms.listProducts).mockResolvedValue({ products: [{ id: 'p1', name: 'Red shoes' } as never], total: 1 })
})

describe('Collections', () => {
  it('lists collections with how they pick products and whether stores show them', async () => {
    renderPage()
    const summer = (await screen.findByText('Summer')).closest('tr')!
    expect(within(summer).getByText('Hand-picked')).toBeInTheDocument()
    expect(within(summer).getByText('Visible')).toBeInTheDocument()
    const sale = screen.getByText('Sale').closest('tr')!
    expect(within(sale).getByText('1 condition (match all)')).toBeInTheDocument()
    expect(within(sale).getByText('Hidden')).toBeInTheDocument()
  })

  it('creates a smart collection with a category condition', async () => {
    vi.mocked(api.createCollection).mockResolvedValue(col({ id: 'new' }))
    renderPage('/collections/new')

    await userEvent.type(await screen.findByLabelText('Name'), 'Summer Shoes')
    expect(screen.getByLabelText('URL')).toHaveValue('summer-shoes')
    await userEvent.click(screen.getByLabelText('Automatically, by conditions'))
    await userEvent.selectOptions(await screen.findByLabelText('Condition 1 category'), 'cat-1')
    await userEvent.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(api.createCollection).toHaveBeenCalledWith('tok', expect.objectContaining({
      name: 'Summer Shoes', slug: 'summer-shoes', type: 'smart', match: 'all', sort: 'newest',
      rules: [{ field: 'category', value: 'cat-1', include_subcategories: true }],
    })))
    expect(api.setCollectionProducts).not.toHaveBeenCalled()
  })

  it('creates a manual collection with hand-picked products', async () => {
    vi.mocked(api.createCollection).mockResolvedValue(col({ id: 'new' }))
    vi.mocked(api.setCollectionProducts).mockResolvedValue(col({ id: 'new' }))
    renderPage('/collections/new')

    await userEvent.type(await screen.findByLabelText('Name'), 'Picks')
    await userEvent.type(screen.getByLabelText('Search products to add'), 'red')
    await userEvent.click(await screen.findByRole('button', { name: 'Red shoes' }))
    await userEvent.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(api.setCollectionProducts).toHaveBeenCalledWith('tok', 'new', ['p1']))
  })

  it('explains what to fix instead of sending an invalid collection', async () => {
    renderPage('/collections/new')
    await userEvent.click(await screen.findByRole('button', { name: 'Save' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('Give the collection a name.')
    expect(api.createCollection).not.toHaveBeenCalled()
  })

  it('opens an existing collection from its URL with product names', async () => {
    vi.mocked(api.getCollection).mockResolvedValue(col({ product_ids: ['p1'] }))
    vi.mocked(pms.getProduct).mockResolvedValue({ product: { name: 'Red shoes' } } as never)
    renderPage('/collections/c1')
    expect(await screen.findByDisplayValue('Summer')).toBeInTheDocument()
    expect(await screen.findByText('Red shoes')).toBeInTheDocument()
  })

  it('deletes after confirmation', async () => {
    vi.mocked(api.deleteCollection).mockResolvedValue()
    renderPage()
    await userEvent.click(await screen.findByRole('button', { name: 'Delete Summer' }))
    await userEvent.click(await screen.findByRole('button', { name: 'Delete collection' }))
    await waitFor(() => expect(api.deleteCollection).toHaveBeenCalledWith('tok', 'c1'))
  })
})
