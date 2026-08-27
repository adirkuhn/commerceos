// @vitest-environment jsdom
import { render, screen } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { ScannerCallbackPage } from '../ScannerCallbackPage'
import { SCANNER_TOKEN_KEY, SCANNER_REFRESH_KEY } from '../useScannerAuth'

function setQuery(query: string) {
  delete (window as unknown as { location: unknown }).location
  // @ts-expect-error -- test-only stub so we can assert on the redirect target
  window.location = { search: query, replace: vi.fn(), origin: 'https://scanner.example.com' }
}

beforeEach(() => {
  localStorage.clear()
})

describe('ScannerCallbackPage', () => {
  it('stores the token+refresh_token under the scanner keys and redirects to /scanner', () => {
    setQuery('?token=a.b.c&refresh_token=refresh-1')

    render(<ScannerCallbackPage />)

    expect(localStorage.getItem(SCANNER_TOKEN_KEY)).toBe('a.b.c')
    expect(localStorage.getItem(SCANNER_REFRESH_KEY)).toBe('refresh-1')
    expect(window.location.replace).toHaveBeenCalledWith('/scanner')
  })

  it('shows an error and does not store anything when refresh_token is missing', () => {
    setQuery('?token=a.b.c')

    render(<ScannerCallbackPage />)

    expect(screen.getByText('No token received from authentication server.')).toBeInTheDocument()
    expect(localStorage.getItem(SCANNER_TOKEN_KEY)).toBeNull()
    expect(window.location.replace).not.toHaveBeenCalled()
  })

  it('shows an error on a malformed token', () => {
    setQuery('?token=not-a-jwt&refresh_token=refresh-1')

    render(<ScannerCallbackPage />)

    expect(screen.getByText('Malformed token received.')).toBeInTheDocument()
    expect(localStorage.getItem(SCANNER_TOKEN_KEY)).toBeNull()
  })
})
