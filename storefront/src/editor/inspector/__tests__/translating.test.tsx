// @vitest-environment jsdom
import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState } from 'react'
import { registry } from '../../../components/registry'
import { PropsForm } from '../PropsForm'

function Translating() {
  const def = registry.get('hero')!
  const props = { ...def.defaults, title: 'Bem-vindo', subtitle: 'Produtos do Brasil', align: 'center' }
  const [translation, setTranslation] = useState<Record<string, unknown>>({ title: 'Welcome' })
  return (
    <>
      <PropsForm def={def} props={props} onChange={() => { throw new Error('default props must not change while translating') }}
        translating={{ translation, onChange: setTranslation }} />
      <output data-testid="translation">{JSON.stringify(translation)}</output>
    </>
  )
}

describe('PropsForm while translating', () => {
  it('edits only the texts, into the translation, starting from the default text', async () => {
    render(<Translating />)

    expect(screen.getByLabelText('Title')).toHaveValue('Welcome')
    expect(screen.getByLabelText('Text')).toHaveValue('Produtos do Brasil')
    expect(screen.queryByLabelText('Alignment')).toBeNull()
    expect(screen.getByText('Shared by every language: switch to the default language to change the other settings.')).toBeInTheDocument()
    expect(screen.getAllByText('Not translated yet')).toHaveLength(2)

    await userEvent.clear(screen.getByLabelText('Text'))
    await userEvent.type(screen.getByLabelText('Text'), 'From Brazil')

    expect(JSON.parse(screen.getByTestId('translation').textContent!)).toEqual({ title: 'Welcome', subtitle: 'From Brazil' })
  })
})
