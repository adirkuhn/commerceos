// @vitest-environment jsdom
import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState } from 'react'
import { PageTextsForm, type PageTexts } from '../PageTextsForm'

function Stateful({ kind, locale, initial }: { kind: 'page' | 'template'; locale: string; initial: PageTexts }) {
  const [texts, setTexts] = useState(initial)
  return <><PageTextsForm kind={kind} texts={texts} locale={locale} onChange={setTexts} /><output data-testid="texts">{JSON.stringify(texts)}</output></>
}

const saved = (): PageTexts => JSON.parse(screen.getByTestId('texts').textContent!)

describe('PageTextsForm', () => {
  it('edits a page\'s title and search texts in the default language', async () => {
    render(<Stateful kind="page" locale="" initial={{ title: 'Sobre', seo: {} }} />)
    await userEvent.type(screen.getByLabelText('Page title'), ' nós')
    await userEvent.type(screen.getByLabelText('Search engine title'), 'Sobre a loja')
    expect(saved()).toEqual({ title: 'Sobre nós', seo: { title: 'Sobre a loja' } })
  })

  it('templates have no title of their own', () => {
    render(<Stateful kind="template" locale="" initial={{ title: '', seo: {} }} />)
    expect(screen.queryByLabelText('Page title')).toBeNull()
    expect(screen.getByLabelText('Search engine description')).toBeInTheDocument()
  })

  it('in another language writes the translation, showing the default as a hint', async () => {
    render(<Stateful kind="page" locale="en" initial={{ title: 'Sobre nós', seo: { title: 'Sobre' } }} />)
    const title = screen.getByLabelText('Page title')
    expect(title).toHaveValue('')
    expect(title).toHaveAttribute('placeholder', 'Sobre nós')
    await userEvent.type(title, 'About us')
    expect(saved()).toEqual({ title: 'Sobre nós', seo: { title: 'Sobre' }, translations: { en: { title: 'About us', seo: {} } } })
  })
})
