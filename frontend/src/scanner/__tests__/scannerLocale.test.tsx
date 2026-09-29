// @vitest-environment jsdom
import { render, screen } from '@testing-library/react'
import i18next from 'i18next'
import { describe, it, expect, vi, afterEach } from 'vitest'
import { PickingDoneScreen } from '../PickingDoneScreen'
import type { MissingPickItem } from '../types'

const missing = [
  { line_item_id: 'l1', name: 'Apple', expected_qty: 10, picked_qty: 6 },
  { line_item_id: 'l2', name: 'Pear', expected_qty: 2, picked_qty: 0 },
] as MissingPickItem[]

describe('scanner texts in pt-BR', () => {
  afterEach(async () => { await i18next.changeLanguage('en') })

  it('the picking done screen reads in Portuguese, with plurals and per-activity wording', async () => {
    await i18next.changeLanguage('pt-BR')
    render(<PickingDoneScreen missingItems={missing} error="" startingPacking={false} onContinueToPacking={vi.fn()} onDone={vi.fn()} />)
    expect(screen.getByRole('heading', { name: 'Pedido separado' })).toBeInTheDocument()
    expect(screen.getByText('2 linhas não foram totalmente separadas:')).toBeInTheDocument()
    expect(screen.getByText('esperado 10 · separado 6')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Continuar para embalagem' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Pronto' })).toBeInTheDocument()
  })
})
