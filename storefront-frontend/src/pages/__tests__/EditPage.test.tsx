// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach } from 'vitest'
import userEvent from '@testing-library/user-event'
import { act, render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import EditPage from '../EditPage'
import type { Block, ThemePage } from '../../types'

vi.mock('../../api/themeDraft', () => ({
  getThemeDraft: vi.fn(),
  saveThemeDraft: vi.fn(),
}))

vi.mock('../../api/client', () => ({
  setStorefrontKey: vi.fn(),
}))

vi.mock('../../api/categories', () => ({
  listCategories: vi.fn().mockResolvedValue([]),
}))

// Real Puck renders a full drag-and-drop editor UI with no simple way to
// trigger onPublish from a test; capturing the prop directly lets the
// round-trip test call it exactly as Puck would on a merchant's publish
// click, without asserting anything about Puck's own internals. Also
// renders overrides.headerActions (EditPage's Save button lives there)
// and stubs usePuck (Save reads Puck's current in-progress data through
// it) so its own real component still works when rendered in these tests.
let capturedOnPublish: ((data: unknown) => void | Promise<void>) | undefined
let mockPuckData: unknown = { content: [], root: {} }
vi.mock('@puckeditor/core', () => ({
  usePuck: () => ({ appState: { data: mockPuckData } }),
  Puck: ({ onPublish, overrides }: {
    onPublish: (data: unknown) => void | Promise<void>
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    overrides?: { headerActions?: (props: { children: any }) => any }
  }) => {
    capturedOnPublish = onPublish
    return (
      <div>
        puck-editor-marker
        {overrides?.headerActions?.({ children: <span>publish-marker</span> })}
      </div>
    )
  },
}))

import * as themeDraftApi from '../../api/themeDraft'
import * as clientApi from '../../api/client'
import * as categoriesApi from '../../api/categories'

function renderAt(pathname: string) {
  return render(
    <MemoryRouter initialEntries={[pathname]}>
      <Routes>
        <Route path="/edit/:slug" element={<EditPage />} />
      </Routes>
    </MemoryRouter>,
  )
}

beforeEach(() => {
  vi.clearAllMocks()
  mockPuckData = { content: [], root: {} }
})

const existingFreeformBlock: Block = {
  id: 'b1', type: 'hero', layout: { desktop: { x: 0, y: 0, width: 100, height: 100 }, mobile: null }, props: {},
}

const draftPage: ThemePage = {
  slug: 'home', canvas_width: 1440, blocks: [existingFreeformBlock], custom_css: 'body { color: red; }',
  editor_mode: 'puck', puck_data: { content: [], root: {} }, sales_channel_id: 'channel-1',
}

const freeformDraftPage: ThemePage = { ...draftPage, editor_mode: 'freeform' }

describe('EditPage', () => {
  it('fetches the draft with the token from the URL, scoped to the slug', async () => {
    vi.mocked(themeDraftApi.getThemeDraft).mockResolvedValue(draftPage)

    renderAt('/edit/home?token=edit-token-1')

    await waitFor(() => expect(themeDraftApi.getThemeDraft).toHaveBeenCalledWith('edit-token-1', 'home'))
  })

  it('scopes the storefront key to this edit session\'s channel, so live blocks fetch the right tenant\'s data', async () => {
    vi.mocked(themeDraftApi.getThemeDraft).mockResolvedValue(draftPage)

    renderAt('/edit/home?token=edit-token-1')

    await waitFor(() => expect(clientApi.setStorefrontKey).toHaveBeenCalledWith('channel-1'))
  })

  it('fetches categories for FeaturedProducts\' picker, non-blocking if it fails', async () => {
    vi.mocked(themeDraftApi.getThemeDraft).mockResolvedValue(draftPage)
    vi.mocked(categoriesApi.listCategories).mockRejectedValue(new Error('network error'))

    renderAt('/edit/home?token=edit-token-1')

    await waitFor(() => expect(categoriesApi.listCategories).toHaveBeenCalled())
    expect(await screen.findByText('puck-editor-marker')).toBeInTheDocument()
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

  it('round-trips the fetched page\'s freeform blocks/custom_css/canvas_width on publish, never wiping them', async () => {
    vi.mocked(themeDraftApi.getThemeDraft).mockResolvedValue(draftPage)
    vi.mocked(themeDraftApi.saveThemeDraft).mockResolvedValue(draftPage)

    renderAt('/edit/home?token=edit-token-1')
    await screen.findByText('puck-editor-marker')

    const newPuckData = { content: [{ type: 'Hero' }], root: {} }
    await act(async () => { await capturedOnPublish!(newPuckData) })

    expect(themeDraftApi.saveThemeDraft).toHaveBeenCalledWith('edit-token-1', 'home', {
      canvas_width: draftPage.canvas_width,
      blocks: draftPage.blocks,
      custom_css: draftPage.custom_css,
      editor_mode: 'puck',
      puck_data: newPuckData,
    })
  })

  it('Publish always sets editor_mode to puck, even starting from freeform', async () => {
    vi.mocked(themeDraftApi.getThemeDraft).mockResolvedValue(freeformDraftPage)
    vi.mocked(themeDraftApi.saveThemeDraft).mockResolvedValue(freeformDraftPage)

    renderAt('/edit/home?token=edit-token-1')
    await screen.findByText('puck-editor-marker')

    await act(async () => { await capturedOnPublish!({ content: [], root: {} }) })

    const [, , payload] = vi.mocked(themeDraftApi.saveThemeDraft).mock.calls[0]
    expect(payload.editor_mode).toBe('puck')
  })

  it('shows a Save button that persists the draft without switching editor_mode away from freeform', async () => {
    const user = userEvent.setup()
    vi.mocked(themeDraftApi.getThemeDraft).mockResolvedValue(freeformDraftPage)
    vi.mocked(themeDraftApi.saveThemeDraft).mockResolvedValue(freeformDraftPage)
    mockPuckData = { content: [{ type: 'Hero' }], root: {} }

    renderAt('/edit/home?token=edit-token-1')
    await screen.findByText('puck-editor-marker')

    await user.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(themeDraftApi.saveThemeDraft).toHaveBeenCalledWith('edit-token-1', 'home', {
      canvas_width: freeformDraftPage.canvas_width,
      blocks: freeformDraftPage.blocks,
      custom_css: freeformDraftPage.custom_css,
      editor_mode: 'freeform',
      puck_data: mockPuckData,
    }))
  })
})
