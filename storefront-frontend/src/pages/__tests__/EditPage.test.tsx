// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import EditPage from '../EditPage'
import type { ThemePage } from '../../types'

vi.mock('../../api/themeDraft', () => ({
  getThemeDraft: vi.fn(),
  saveThemeDraft: vi.fn(),
}))

import * as themeDraftApi from '../../api/themeDraft'

function renderAt(pathname: string) {
  return render(
    <MemoryRouter initialEntries={[pathname]}>
      <Routes>
        <Route path="/edit/:slug" element={<EditPage />} />
      </Routes>
    </MemoryRouter>,
  )
}

beforeEach(() => { vi.clearAllMocks() })

const draftPage: ThemePage = {
  slug: 'home', canvas_width: 1440, blocks: [], custom_css: '',
  editor_mode: 'puck', puck_data: { content: [], root: {} },
}

describe('EditPage', () => {
  it('fetches the draft with the token from the URL, scoped to the slug', async () => {
    vi.mocked(themeDraftApi.getThemeDraft).mockResolvedValue(draftPage)

    renderAt('/edit/home?token=edit-token-1')

    await waitFor(() => expect(themeDraftApi.getThemeDraft).toHaveBeenCalledWith('edit-token-1', 'home'))
  })

  it('shows an explanatory error when opened with no token', async () => {
    renderAt('/edit/home')

    expect(await screen.findByText(/Missing edit session token/)).toBeInTheDocument()
    expect(themeDraftApi.getThemeDraft).not.toHaveBeenCalled()
  })

  it('shows the server error message when the token is invalid or expired', async () => {
    vi.mocked(themeDraftApi.getThemeDraft).mockRejectedValue(new Error('invalid edit session'))

    renderAt('/edit/home?token=wrong-token')

    expect(await screen.findByText(/invalid edit session/)).toBeInTheDocument()
  })
})
