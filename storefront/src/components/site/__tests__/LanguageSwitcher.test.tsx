// @vitest-environment jsdom
import { describe, it, expect, afterEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { ReactNode } from 'react'
import { rendererFor } from '../../renderers'
import { RenderProvider, type RenderEnv } from '../../../runtime/RenderContext'
import { StoreLanguageProvider, type StoreLanguageValue } from '../../../runtime/page/StoreLanguageContext'
import { SHOP_LANGUAGE_COOKIE } from '../../../runtime/storeLocale'

const Switcher = rendererFor('language-switcher')!
const node = { id: 'ls', type: 'language-switcher' }

const languages = (over: Partial<StoreLanguageValue> = {}): StoreLanguageValue => ({
  storeKey: 'k1', locale: 'pt-BR', offered: ['pt-BR', 'en', 'nl'],
  hrefFor: l => (l === 'pt-BR' ? '/store/k1/products/x' : `/store/k1/${l}/products/x`),
  ...over,
})

function renderWith(el: ReactNode, value?: StoreLanguageValue, env: Partial<RenderEnv> = {}) {
  const inner = value ? <StoreLanguageProvider value={value}>{el}</StoreLanguageProvider> : el
  return render(<RenderProvider value={{ storePath: p => `/store/k1/${p}`, ...env }}>{inner}</RenderProvider>)
}

afterEach(() => {
  window.history.pushState({}, '', '/store/k1/')
  document.cookie = `${SHOP_LANGUAGE_COOKIE}=; Path=/store/k1/; Max-Age=0`
  window.history.pushState({}, '', '/')
})

describe('language switcher', () => {
  it('links the current page in every offered language, marking the current one', () => {
    renderWith(<Switcher node={node} props={{ display: 'names' }} className="n-ls" />, languages())

    const nav = screen.getByRole('navigation')
    expect(screen.getByRole('link', { name: 'English' })).toHaveAttribute('href', '/store/k1/en/products/x')
    expect(screen.getByRole('link', { name: 'English' })).toHaveAttribute('hreflang', 'en')
    expect(screen.getByRole('link', { name: 'Nederlands' })).toHaveAttribute('href', '/store/k1/nl/products/x')
    expect(screen.getByRole('link', { name: 'Português (Brasil)' })).toHaveAttribute('aria-current', 'true')
    expect(nav).toHaveAttribute('aria-label', 'Language')
  })

  it('can show short codes', () => {
    renderWith(<Switcher node={node} props={{ display: 'codes' }} className="" />, languages())
    expect(screen.getByRole('link', { name: 'English' })).toHaveTextContent('EN')
    expect(screen.getByRole('link', { name: 'Português (Brasil)' })).toHaveTextContent('PT')
  })

  it('remembers the pick for next time, for this store only', async () => {
    window.history.pushState({}, '', '/store/k1/products/x')
    renderWith(<Switcher node={node} props={{ display: 'names' }} className="" />, languages({ hrefFor: () => '#' }))
    await userEvent.click(screen.getByRole('link', { name: 'Nederlands' }))
    expect(document.cookie).toContain(`${SHOP_LANGUAGE_COOKIE}=nl`)
    window.history.pushState({}, '', '/')
    expect(document.cookie).not.toContain(SHOP_LANGUAGE_COOKIE)
  })

  it('shows nothing while the store offers one language', () => {
    const { container } = renderWith(<Switcher node={node} props={{ display: 'names' }} className="" />, languages({ offered: ['pt-BR'] }))
    expect(container).toBeEmptyDOMElement()
  })

  it('shows a hint in the editor, where there is no store page to switch', () => {
    renderWith(<Switcher node={node} props={{ display: 'names' }} className="" />, undefined, { mode: 'editor' })
    expect(screen.getByText(/more than one language/)).toBeInTheDocument()
  })
})
