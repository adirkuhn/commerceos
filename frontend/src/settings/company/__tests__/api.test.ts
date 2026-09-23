import { describe, it, expect, vi, beforeEach } from 'vitest'
import { getCompanyProfile, updateCompanyProfile, CompanyProfileError } from '../api'
import type { CompanyProfile, CompanyProfileInput } from '../types'

const TOKEN = 'test-token'
const BASE = 'https://api.commerceos.dev.localhost/api/v1/settings/company-profile'

const input: CompanyProfileInput = {
  legal_name: 'Cantinho B.V.', trading_name: '', vat_number: 'NL123456789B01', chamber_of_commerce: '',
  email: '', phone: '', iban: '',
  address: { street: '', house_number: '', postal_code: '', city: '', country: 'NL' },
}
const profile: CompanyProfile = { ...input, updated_at: '2026-09-24T10:00:00Z' }

function respond(status: number, body: unknown) {
  const fetchMock = vi.fn().mockResolvedValue({ ok: status < 300, status, json: () => Promise.resolve(body) })
  vi.stubGlobal('fetch', fetchMock)
  return fetchMock
}

beforeEach(() => { vi.restoreAllMocks() })

describe('company profile api', () => {
  it('GETs the profile with the bearer token', async () => {
    const fetchMock = respond(200, { data: profile, error: null })

    expect(await getCompanyProfile(TOKEN)).toEqual(profile)
    expect(fetchMock).toHaveBeenCalledWith(BASE, expect.objectContaining({
      method: 'GET', headers: expect.objectContaining({ Authorization: `Bearer ${TOKEN}` }),
    }))
  })

  it('PUTs the whole profile', async () => {
    const fetchMock = respond(200, { data: profile, error: null })

    await updateCompanyProfile(TOKEN, input)

    expect(fetchMock).toHaveBeenCalledWith(BASE, expect.objectContaining({ method: 'PUT', body: JSON.stringify(input) }))
  })

  it('rejects with the error code and the field it belongs to', async () => {
    respond(422, { data: null, error: { code: 'INVALID_IBAN', message: 'IBAN is not valid' } })

    const err = await updateCompanyProfile(TOKEN, input).catch(e => e)

    expect(err).toBeInstanceOf(CompanyProfileError)
    expect(err.code).toBe('INVALID_IBAN')
    expect(err.field).toBe('iban')
    expect(err.message).toBe('IBAN is not valid')
  })

  it('reports errors without a field (e.g. no permission) with field undefined', async () => {
    respond(403, { data: null, error: { code: 'FORBIDDEN', message: 'insufficient role' } })

    const err = await updateCompanyProfile(TOKEN, input).catch(e => e)

    expect(err.code).toBe('FORBIDDEN')
    expect(err.field).toBeUndefined()
  })
})
