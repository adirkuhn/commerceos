// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'

vi.mock('../quotesApi', () => ({ getQuote: vi.fn(), listQuotes: vi.fn() }))
import * as api from '../quotesApi'
import { useSavedQuote, useSavedQuotes } from '../hooks/useSavedQuotes'

beforeEach(() => vi.clearAllMocks())

describe('useSavedQuote', () => {
  it('loads the open quote', async () => {
    vi.mocked(api.getQuote).mockResolvedValue({ id: 'q-1' } as never)
    const { result } = renderHook(() => useSavedQuote('tok', 'q-1'))
    await waitFor(() => expect(result.current.data).toEqual({ id: 'q-1' }))
    expect(api.getQuote).toHaveBeenCalledWith('tok', 'q-1')
  })

  it('loads nothing for an unsaved quote', () => {
    renderHook(() => useSavedQuote('tok', null))
    expect(api.getQuote).not.toHaveBeenCalled()
  })
})

describe('useSavedQuotes', () => {
  it('lists with the query', async () => {
    vi.mocked(api.listQuotes).mockResolvedValue({ quotes: [], total: 0 })
    const { result } = renderHook(() => useSavedQuotes('tok', { page: 2, perPage: 25, search: 'sab' }))
    await waitFor(() => expect(result.current.data).toEqual({ quotes: [], total: 0 }))
    expect(api.listQuotes).toHaveBeenCalledWith('tok', { page: 2, perPage: 25, search: 'sab' })
  })
})
