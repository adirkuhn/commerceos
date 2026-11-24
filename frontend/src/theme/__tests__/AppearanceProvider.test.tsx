// @vitest-environment jsdom
import { act, render, screen, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { DEFAULT_APPEARANCE, type Appearance } from '../appearance'
import { APPEARANCE_STORAGE_KEY } from '../applyAppearance'
import { LOOK_STORAGE_KEY } from '../lookPreference'
import { setShopDefaultLook } from '../useLook'

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
const htmlClasses = () => document.documentElement.classList

beforeEach(() => {
  vi.clearAllMocks()
  document.head.innerHTML = ''
  localStorage.clear()
  document.documentElement.className = ''
  vi.mocked(api.getAppearance).mockResolvedValue(saved)
})

afterEach(() => { setShopDefaultLook(null) })

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

  it('shows the shop default look to someone without a look of their own', async () => {
    vi.mocked(api.getAppearance).mockResolvedValue({ ...saved, default_look: 'graphite' })
    mount()
    await screen.findByText('loaded')

    expect(htmlClasses()).toContain('look-graphite')
    expect(htmlClasses()).not.toContain('look-wallet')
    expect(JSON.parse(localStorage.getItem(APPEARANCE_STORAGE_KEY)!).look).toBe('graphite')
  })

  it('keeps a person\'s own look over the shop default', async () => {
    localStorage.setItem(LOOK_STORAGE_KEY, 'wallet')
    vi.mocked(api.getAppearance).mockResolvedValue({ ...saved, default_look: 'graphite' })
    mount()
    await screen.findByText('loaded')

    expect(htmlClasses()).toContain('look-wallet')
  })

  it('previews a new default look and goes back to the saved one', async () => {
    mount()
    await screen.findByText('loaded')

    act(() => state.preview({ ...DEFAULT_APPEARANCE, default_look: 'graphite' }))
    expect(htmlClasses()).toContain('look-graphite')

    act(() => state.preview(null))
    expect(htmlClasses()).toContain('look-wallet')
    expect(JSON.parse(localStorage.getItem(APPEARANCE_STORAGE_KEY)!).look).toBe('wallet')
  })

  it('applies a saved default look', async () => {
    vi.mocked(api.updateAppearance).mockResolvedValue({ ...saved, default_look: 'graphite' })
    mount()
    await screen.findByText('loaded')

    await act(async () => { await state.save({ ...DEFAULT_APPEARANCE, default_look: 'graphite' }) })

    expect(htmlClasses()).toContain('look-graphite')
  })

  it('says so when the appearance cannot load', async () => {
    vi.mocked(api.getAppearance).mockRejectedValue(new Error('boom'))
    mount()

    await waitFor(() => expect(screen.getByText('boom')).toBeInTheDocument())
  })
})
