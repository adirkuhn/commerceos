// @vitest-environment jsdom
import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import Reviews from '../Reviews'
import { renderAt } from '../../test/renderAt'
import type { Review } from '../../reviews/types'

vi.mock('../../auth/AuthContext', () => ({
  useAuth: () => ({ token: 'test-token', user: null, login: vi.fn(), logout: vi.fn(), isAuthenticated: true }),
}))

vi.mock('../../reviews/api', () => ({
  listReviews: vi.fn(),
  deleteReview: vi.fn(),
}))

import * as reviewsApi from '../../reviews/api'

function review(overrides: Partial<Review> = {}): Review {
  return {
    id: 'review-1', sales_channel_id: 'channel-1', product_id: 'product-1',
    customer_name: 'Ada', customer_email: 'ada@example.com', rating: 5,
    title: 'Great', body: 'Loved it', verified: true, created_at: '2026-06-01T00:00:00Z',
    ...overrides,
  }
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(reviewsApi.listReviews).mockResolvedValue([])
})

describe('Reviews', () => {
  it('shows an empty state when there are no reviews', async () => {
    renderAt(<Reviews />, '/reviews')

    expect(await screen.findByText('No reviews yet')).toBeInTheDocument()
  })

  it('lists reviews with customer, rating, and body', async () => {
    vi.mocked(reviewsApi.listReviews).mockResolvedValue([review()])

    renderAt(<Reviews />, '/reviews')

    expect(await screen.findByText('Ada')).toBeInTheDocument()
    expect(screen.getByText('Loved it')).toBeInTheDocument()
    expect(screen.getByLabelText('5 out of 5 stars')).toBeInTheDocument()
  })

  it('filters by search', async () => {
    vi.mocked(reviewsApi.listReviews).mockResolvedValue([
      review(), review({ id: 'review-2', customer_name: 'Bob', body: 'Not great' }),
    ])

    renderAt(<Reviews />, '/reviews')
    await screen.findByText('Ada')

    await userEvent.type(screen.getByPlaceholderText('Search by customer, product, or text...'), 'Bob')

    expect(screen.queryByText('Ada')).not.toBeInTheDocument()
    expect(screen.getByText('Bob')).toBeInTheDocument()
  })

  it('deleting a review calls deleteReview', async () => {
    vi.mocked(reviewsApi.listReviews).mockResolvedValue([review()])
    vi.stubGlobal('confirm', vi.fn(() => true))

    renderAt(<Reviews />, '/reviews')
    await userEvent.click(await screen.findByTitle('Delete'))

    await waitFor(() => expect(reviewsApi.deleteReview).toHaveBeenCalledWith('test-token', 'review-1'))
  })
})

describe('Reviews URLs', () => {
  it('search (?q=) and page (?page=) are read from the URL', async () => {
    vi.mocked(reviewsApi.listReviews).mockResolvedValue(Array.from({ length: 25 }, (_, i) => review({ id: `r${i}` })))
    renderAt(<Reviews />, '/reviews', '/reviews?page=2')
    expect(await screen.findByText('Page 2 of 2')).toBeInTheDocument()
  })

  it('typing a search writes ?q= and drops the page', async () => {
    vi.mocked(reviewsApi.listReviews).mockResolvedValue(Array.from({ length: 25 }, (_, i) => review({ id: `r${i}` })))
    const { router } = renderAt(<Reviews />, '/reviews', '/reviews?page=2')
    await userEvent.type(await screen.findByPlaceholderText('Search by customer, product, or text...'), 'x')
    expect(router.state.location.search).toBe('?q=x')
  })
})
