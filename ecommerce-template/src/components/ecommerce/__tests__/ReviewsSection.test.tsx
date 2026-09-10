// @vitest-environment jsdom
import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'
import { ReviewsSection } from '../ReviewsSection'
import { renderWithProviders } from '@/test/testUtils'
import { PRODUCTS, REVIEWS } from '@/config/demoData'

const productWithReviews = PRODUCTS.find(p => REVIEWS.some(r => r.productId === p.id))!
// MockReviewRepository's submitted-review store is module-level and
// persists for this whole test file -- each test below that submits (or
// asserts an empty state) uses its own distinct never-reviewed product so
// they can't contaminate each other.
const noReviewProducts = PRODUCTS.filter(p => !REVIEWS.some(r => r.productId === p.id))

describe('ReviewsSection', () => {
  it('shows the existing reviews and average rating for a reviewed product', async () => {
    renderWithProviders(<ReviewsSection productId={productWithReviews.id} />)
    await waitFor(() => expect(screen.queryByText('No reviews yet.')).not.toBeInTheDocument())
    const reviewCount = REVIEWS.filter(r => r.productId === productWithReviews.id).length
    expect(await screen.findByText(`Based on ${reviewCount} reviews`)).toBeInTheDocument()
  })

  it('shows an empty state for a product with no reviews yet', async () => {
    renderWithProviders(<ReviewsSection productId={noReviewProducts[0].id} />)
    expect(await screen.findByText(/Be the first to share your thoughts/)).toBeInTheDocument()
  })

  it('submits a new review and it appears in the list', async () => {
    const user = userEvent.setup()
    renderWithProviders(<ReviewsSection productId={noReviewProducts[1].id} />)
    await screen.findByText(/Be the first to share your thoughts/)

    await user.click(screen.getByRole('button', { name: 'Write a review' }))
    await user.type(screen.getByLabelText('Your name'), 'Riley P.')
    await user.click(screen.getByRole('button', { name: '4 star' }))
    await user.type(screen.getByLabelText('Title'), 'Great everyday item')
    await user.type(screen.getByLabelText('Review'), 'Held up well over a full season of regular use.')
    await user.click(screen.getByRole('button', { name: 'Submit review' }))

    expect(await screen.findByText('Great everyday item')).toBeInTheDocument()
    expect(screen.getByText('Held up well over a full season of regular use.')).toBeInTheDocument()
  })

  it('requires a title and a body of at least 10 characters', async () => {
    const user = userEvent.setup()
    renderWithProviders(<ReviewsSection productId={noReviewProducts[2].id} />)
    await screen.findByText(/Be the first to share your thoughts/)

    await user.click(screen.getByRole('button', { name: 'Write a review' }))
    await user.type(screen.getByLabelText('Review'), 'too short')
    await user.click(screen.getByRole('button', { name: 'Submit review' }))

    expect(await screen.findByText('Give your review a title')).toBeInTheDocument()
    expect(screen.getByText('Reviews need at least 10 characters')).toBeInTheDocument()
  })
})
