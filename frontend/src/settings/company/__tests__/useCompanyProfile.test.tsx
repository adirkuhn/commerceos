// @vitest-environment jsdom
import { act, renderHook, waitFor } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { useCompanyProfile } from '../useCompanyProfile'
import { CompanyProfileError } from '../api'
import type { CompanyProfile } from '../types'

vi.mock('../api', async importOriginal => {
  const actual = await importOriginal<typeof import('../api')>()
  return { ...actual, getCompanyProfile: vi.fn(), updateCompanyProfile: vi.fn() }
})
import * as api from '../api'

const saved: CompanyProfile = {
  legal_name: 'Cantinho B.V.', trading_name: '', vat_number: '', chamber_of_commerce: '', email: '', phone: '', iban: '',
  address: { street: '', house_number: '', postal_code: '', city: '', country: 'NL' }, updated_at: '2026-09-24T10:00:00Z',
}

beforeEach(() => { vi.clearAllMocks() })

describe('useCompanyProfile', () => {
  it('loads the profile', async () => {
    vi.mocked(api.getCompanyProfile).mockResolvedValue(saved)
    const { result } = renderHook(() => useCompanyProfile('tok'))

    expect(result.current.loading).toBe(true)
    await waitFor(() => expect(result.current.loading).toBe(false))
    expect(result.current.profile).toEqual(saved)
    expect(api.getCompanyProfile).toHaveBeenCalledWith('tok')
  })

  it('exposes a load error', async () => {
    vi.mocked(api.getCompanyProfile).mockRejectedValue(new CompanyProfileError('INTERNAL', 'boom'))
    const { result } = renderHook(() => useCompanyProfile('tok'))

    await waitFor(() => expect(result.current.error).toBe('boom'))
    expect(result.current.profile).toBeNull()
  })

  it('save replaces the profile with what the server returns', async () => {
    vi.mocked(api.getCompanyProfile).mockResolvedValue(saved)
    vi.mocked(api.updateCompanyProfile).mockResolvedValue({ ...saved, legal_name: 'NEW B.V.' })
    const { result } = renderHook(() => useCompanyProfile('tok'))
    await waitFor(() => expect(result.current.loading).toBe(false))

    await act(() => result.current.save({ ...saved, legal_name: 'new b.v.' }))

    expect(result.current.profile?.legal_name).toBe('NEW B.V.')
  })

  it('save rethrows so the form can place the error', async () => {
    vi.mocked(api.getCompanyProfile).mockResolvedValue(saved)
    vi.mocked(api.updateCompanyProfile).mockRejectedValue(new CompanyProfileError('INVALID_IBAN', 'bad'))
    const { result } = renderHook(() => useCompanyProfile('tok'))
    await waitFor(() => expect(result.current.loading).toBe(false))

    await expect(result.current.save(saved)).rejects.toBeInstanceOf(CompanyProfileError)
    expect(result.current.profile).toEqual(saved)
  })
})
