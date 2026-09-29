// @vitest-environment jsdom
import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { ContentLanguageSelect } from '../ContentLanguageSelect'

describe('ContentLanguageSelect', () => {
  it('switches which language the texts are edited in, the default first', async () => {
    const onChange = vi.fn()
    render(<ContentLanguageSelect languages={{ defaultLocale: 'pt-BR', offered: ['pt-BR', 'en', 'nl'] }} value="" onChange={onChange} />)

    const select = screen.getByLabelText('Texts in')
    expect([...select.querySelectorAll('option')].map(o => o.textContent)).toEqual(['Português (Brasil) (default)', 'English', 'Nederlands'])
    await userEvent.selectOptions(select, 'en')
    expect(onChange).toHaveBeenCalledWith('en')
  })

  it('is hidden while the store offers one language', () => {
    const { container } = render(<ContentLanguageSelect languages={{ defaultLocale: 'en', offered: ['en'] }} value="" onChange={vi.fn()} />)
    expect(container).toBeEmptyDOMElement()
  })
})
