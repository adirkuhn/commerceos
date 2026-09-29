// @vitest-environment jsdom
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import i18next from 'i18next'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { LoginPage } from '../LoginPage'

function mockFetch(data: unknown, ok = true) {
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
    ok, json: () => Promise.resolve({ data, error: ok ? null : { message: 'Google sign-in is unavailable' } }),
  }))
}

beforeEach(() => {
  vi.restoreAllMocks()
  vi.stubEnv('VITE_LM2_LOGIN_URL', 'https://lm2.example.com/login')
  delete (window as unknown as { location: unknown }).location
  // @ts-expect-error -- test-only stub so we can assert on the redirect target
  window.location = { href: '', origin: 'https://app.commerceos.example.com' }
})

describe('LoginPage — Google sign-in', () => {
  it('fetches the google login url from lm2 and redirects to it', async () => {
    mockFetch({ url: 'https://accounts.google.com/o/oauth2/v2/auth?state=abc' })
    render(<LoginPage />)

    fireEvent.click(screen.getByRole('button', { name: /sign in with google/i }))

    await waitFor(() => expect(window.location.href).toBe('https://accounts.google.com/o/oauth2/v2/auth?state=abc'))
    const [url] = vi.mocked(fetch).mock.calls[0]
    expect(url).toContain('https://lm2.example.com/api/v1/auth/app/google/url')
    expect(url).toContain('app=commerceos')
  })

  it('shows an error when lm2 has no Google configured', async () => {
    mockFetch(null, false)
    render(<LoginPage />)

    fireEvent.click(screen.getByRole('button', { name: /sign in with google/i }))

    expect(await screen.findByRole('alert')).toHaveTextContent('Google sign-in is unavailable')
  })
})

describe('LoginPage — pt-BR', () => {
  it('speaks Portuguese when the admin language is pt-BR', async () => {
    await i18next.changeLanguage('pt-BR')
    try {
      render(<LoginPage />)
      expect(screen.getByText('Entre para continuar')).toBeInTheDocument()
      expect(screen.getByRole('button', { name: 'Entrar com Google' })).toBeInTheDocument()
    } finally {
      await i18next.changeLanguage('en')
    }
  })
})
