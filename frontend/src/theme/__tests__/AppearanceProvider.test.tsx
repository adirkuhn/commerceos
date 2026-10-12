// @vitest-environment jsdom
import { act, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { DEFAULT_APPEARANCE, type Appearance } from '../appearance'
import { APPEARANCE_STORAGE_KEY } from '../applyAppearance'

vi.mock('../../settings/appearance/api', () => ({ getAppearance: vi.fn(), updateAppearance: vi.fn() }))

import * as api from '../../settings/appearance/api'
import { AppearanceProvider, useAppearance, type AppearanceState } from '../AppearanceProvider'

const saved: Appearance = { ...DEFAULT_APPEARANCE, corners: 'round', updated_at: '2026-10-12T10:00:00Z' }
let state: AppearanceState

function Probe() {
  state = useAppearance()
  return <p>{state.error || (state.appearance ? 'loaded' : 'loading')}</p>
}

function mount() {
  render(<AppearanceProvider token="t"><Probe /></AppearanceProvider>)
}

const css = () => document.getElementById('appearance-css')?.textContent ?? ''

beforeEach(() => {
  vi.clearAllMocks()
  document.head.innerHTML = ''
  localStorage.clear()
  vi.mocked(api.getAppearance).mockResolvedValue(saved)
})

describe('AppearanceProvider', () => {
  it('applies the shop appearance and remembers it for the next first paint', async () => {
    mount()

    expect(await screen.findByText('loaded')).toBeInTheDocument()
    expect(css()).toContain('--radius-lg: 10px;')
    expect(localStorage.getItem(APPEARANCE_STORAGE_KEY)).toContain('--radius-lg: 10px;')
  })

  it('previews a look without remembering it, then goes back to the saved one', async () => {
    mount()
    await screen.findByText('loaded')

    act(() => state.preview({ ...DEFAULT_APPEARANCE, corners: 'sharp' }))
    expect(css()).toContain('--radius-lg: 3px;')
    expect(localStorage.getItem(APPEARANCE_STORAGE_KEY)).not.toContain('3px')

    act(() => state.preview(null))
    expect(css()).toContain('--radius-lg: 10px;')
  })

  it('saves, applies and remembers the new look', async () => {
    const next: Appearance = { ...DEFAULT_APPEARANCE, density: 'compact', updated_at: '2026-10-12T11:00:00Z' }
    vi.mocked(api.updateAppearance).mockResolvedValue(next)
    mount()
    await screen.findByText('loaded')

    await act(async () => { await state.save({ ...DEFAULT_APPEARANCE, density: 'compact' }) })

    expect(api.updateAppearance).toHaveBeenCalledWith('t', { ...DEFAULT_APPEARANCE, density: 'compact' })
    expect(css()).toContain('--density: 0.8;')
    expect(localStorage.getItem(APPEARANCE_STORAGE_KEY)).toContain('--density: 0.8;')
    expect(state.appearance).toEqual(next)
  })

  it('says so when the appearance cannot load', async () => {
    vi.mocked(api.getAppearance).mockRejectedValue(new Error('boom'))
    mount()

    await waitFor(() => expect(screen.getByText('boom')).toBeInTheDocument())
  })
})
