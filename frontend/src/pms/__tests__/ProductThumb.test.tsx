// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import { ProductThumb, ADMIN_THUMB, SCANNER_THUMB } from '../ProductThumb'
import { useProductMedia } from '../hooks/useProductMedia'

vi.mock('../hooks/useProductMedia', () => ({ useProductMedia: vi.fn() }))

const mockMedia = (media: { url: string; alt_text: string }[]) =>
  vi.mocked(useProductMedia).mockReturnValue({ media } as unknown as ReturnType<typeof useProductMedia>)

describe('ProductThumb', () => {
  beforeEach(() => vi.mocked(useProductMedia).mockReset())

  it("shows the product's featured image", () => {
    mockMedia([{ url: 'https://cdn/x.jpg', alt_text: 'Front' }, { url: 'https://cdn/y.jpg', alt_text: 'Back' }])
    render(<ProductThumb token="t" productId="p1" appearance={ADMIN_THUMB} />)
    expect(screen.getByAltText('Front')).toHaveAttribute('src', 'https://cdn/x.jpg')
    expect(useProductMedia).toHaveBeenCalledWith('t', 'p1')
  })

  it('falls back to a placeholder when the line has no product or media', () => {
    mockMedia([])
    const { container } = render(<ProductThumb token="t" appearance={SCANNER_THUMB} title="Apples" />)
    expect(screen.queryByRole('img')).not.toBeInTheDocument()
    expect(useProductMedia).toHaveBeenCalledWith('t', null)
    expect(container.firstChild).toHaveAttribute('title', 'Apples')
    expect(container.querySelector('svg')).toBeInTheDocument()
  })

  it('applies the appearance preset', () => {
    mockMedia([])
    const { container } = render(<ProductThumb token="t" appearance={SCANNER_THUMB} />)
    expect(container.firstChild).toHaveClass('w-9', 'h-9', 'bg-night-700')
  })
})
