// @vitest-environment jsdom
import { renderHook, act, waitFor } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { useMoneybirdSetup } from '../useMoneybirdSetup'
import { CodedApiError } from '../../../api/codedApiError'
import type { MoneybirdSetup } from '../../types'

vi.mock('../../api', () => ({ getMoneybirdSetup: vi.fn(), updateMoneybirdSettings: vi.fn() }))
import * as api from '../../api'

const setup: MoneybirdSetup = {
  settings: {
    connection_id: 'conn-1', administration_id: 'adm-1', administration_name: 'Cantinho', default_ledger_account_id: '',
    auto_sync: false, tax_rates: [], payment_methods: [],
  },
  administrations: [{ id: 'adm-1', name: 'Cantinho', currency: 'EUR' }],
  tax_rates: [], ledger_accounts: [], payment_methods: ['cash', 'bank_transfer'],
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(api.getMoneybirdSetup).mockResolvedValue(setup)
})

describe('useMoneybirdSetup', () => {
  it('loads the setup', async () => {
    const { result } = renderHook(() => useMoneybirdSetup('t'))
    await waitFor(() => expect(result.current.setup).toEqual(setup))
    expect(result.current.loading).toBe(false)
  })

  it('shows why loading failed', async () => {
    vi.mocked(api.getMoneybirdSetup).mockRejectedValue(new CodedApiError('MONEYBIRD_TOKEN_INVALID', 'x', undefined))
    const { result } = renderHook(() => useMoneybirdSetup('t'))
    await waitFor(() => expect(result.current.error).toBe("Moneybird didn't accept the API token."))
  })

  // Choosing an administration changes the lists (tax rates, ledger accounts), so the setup reloads.
  it('saves only the fields given, then reloads', async () => {
    vi.mocked(api.updateMoneybirdSettings).mockResolvedValue(setup.settings)
    const { result } = renderHook(() => useMoneybirdSetup('t'))
    await waitFor(() => expect(result.current.setup).not.toBeNull())

    await act(() => result.current.save({ auto_sync: true }))

    expect(api.updateMoneybirdSettings).toHaveBeenCalledWith('t', { auto_sync: true })
    expect(api.getMoneybirdSetup).toHaveBeenCalledTimes(2)
  })

  it('save rejects so the section can show why', async () => {
    vi.mocked(api.updateMoneybirdSettings).mockRejectedValue(new CodedApiError('MONEYBIRD_UNKNOWN_ID', 'x', undefined))
    const { result } = renderHook(() => useMoneybirdSetup('t'))
    await waitFor(() => expect(result.current.setup).not.toBeNull())
    await expect(result.current.save({ default_ledger_account_id: 'x' })).rejects.toBeInstanceOf(CodedApiError)
  })
})
