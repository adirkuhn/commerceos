// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { WishlistProvider, useWishlist } from '../WishlistContext'
import type { Wishlist } from '../../types'

vi.mock('../../api/wishlist', () => ({
  createWishlist: vi.fn(),
  getWishlist: vi.fn(),
  toggleWishlistItem: vi.fn(),
}))

import * as wishlistApi from '../../api/wishlist'

const STORAGE_KEY = 'commerceos_wishlist_token'

function Harness() {
  const { wishlist, isSaved, toggle, error } = useWishlist()
  return (
    <div>
      <p data-testid="ids">{wishlist ? JSON.stringify(wishlist.product_ids) : 'no wishlist'}</p>
      <p data-testid="saved">{isSaved('p1') ? 'saved' : 'not saved'}</p>
      {error && <p data-testid="error">{error}</p>}
      <button onClick={() => { toggle('p1').catch(() => {}) }}>Toggle p1</button>
    </div>
  )
}

beforeEach(() => {
  vi.clearAllMocks()
  localStorage.clear()
})

describe('WishlistContext', () => {
  it('starts empty with no stored token', async () => {
    render(<WishlistProvider><Harness /></WishlistProvider>)
    await waitFor(() => expect(screen.getByTestId('ids')).toHaveTextContent('no wishlist'))
    expect(wishlistApi.getWishlist).not.toHaveBeenCalled()
  })

  it('creates a wishlist lazily on first toggle and persists the token', async () => {
    const user = userEvent.setup()
    const created: Wishlist = { token: 'wishlist-1', product_ids: [] }
    const updated: Wishlist = { token: 'wishlist-1', product_ids: ['p1'] }
    vi.mocked(wishlistApi.createWishlist).mockResolvedValue(created)
    vi.mocked(wishlistApi.toggleWishlistItem).mockResolvedValue(updated)

    render(<WishlistProvider><Harness /></WishlistProvider>)
    await waitFor(() => expect(screen.getByTestId('ids')).toHaveTextContent('no wishlist'))

    await user.click(screen.getByText('Toggle p1'))

    await waitFor(() => expect(screen.getByTestId('saved')).toHaveTextContent('saved'))
    expect(wishlistApi.toggleWishlistItem).toHaveBeenCalledWith('wishlist-1', 'p1')
    expect(localStorage.getItem(STORAGE_KEY)).toBe('wishlist-1')
  })

  it('loads the existing wishlist on mount when a token is already stored', async () => {
    localStorage.setItem(STORAGE_KEY, 'wishlist-9')
    vi.mocked(wishlistApi.getWishlist).mockResolvedValue({ token: 'wishlist-9', product_ids: ['p1'] })

    render(<WishlistProvider><Harness /></WishlistProvider>)

    await waitFor(() => expect(screen.getByTestId('saved')).toHaveTextContent('saved'))
    expect(wishlistApi.getWishlist).toHaveBeenCalledWith('wishlist-9')
  })

  it('surfaces an error from a rejected toggle', async () => {
    const user = userEvent.setup()
    localStorage.setItem(STORAGE_KEY, 'wishlist-9')
    vi.mocked(wishlistApi.getWishlist).mockResolvedValue({ token: 'wishlist-9', product_ids: [] })
    vi.mocked(wishlistApi.toggleWishlistItem).mockRejectedValue(new Error('product not available on this sales channel'))

    render(<WishlistProvider><Harness /></WishlistProvider>)
    await waitFor(() => expect(screen.getByTestId('saved')).toHaveTextContent('not saved'))

    await user.click(screen.getByText('Toggle p1'))

    await waitFor(() => expect(screen.getByTestId('error')).toHaveTextContent('product not available on this sales channel'))
  })
})
