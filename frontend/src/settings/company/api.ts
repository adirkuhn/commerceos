import type { CompanyProfile, CompanyProfileField, CompanyProfileInput } from './types'

function url(): string {
  return `${import.meta.env.VITE_API_BASE_URL}/settings/company-profile`
}

// Backend error code -> the form field it belongs to (see mapError in
// backend/infrastructure/handlers/settings).
const FIELD_BY_CODE: Readonly<Record<string, CompanyProfileField>> = {
  LEGAL_NAME_REQUIRED: 'legal_name',
  INVALID_EMAIL: 'email',
  INVALID_VAT_NUMBER: 'vat_number',
  INVALID_IBAN: 'iban',
  INVALID_COUNTRY_CODE: 'address.country',
}

// Keeps the API's error code (and the field it points at) so the form can
// show the message next to the right input.
export class CompanyProfileError extends Error {
  readonly field: CompanyProfileField | undefined

  constructor(readonly code: string, message: string) {
    super(message)
    this.name = 'CompanyProfileError'
    this.field = FIELD_BY_CODE[code]
  }
}

interface Envelope<T> {
  data: T | null
  error: { code: string; message: string } | null
}

async function send<T>(method: 'GET' | 'PUT', token: string, body?: unknown): Promise<T> {
  const res = await fetch(url(), {
    method,
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
    ...(body === undefined ? {} : { body: JSON.stringify(body) }),
  })
  const envelope: Envelope<T> = await res.json()
  if (!res.ok || envelope.error) {
    throw new CompanyProfileError(envelope.error?.code ?? 'HTTP_ERROR', envelope.error?.message ?? `HTTP ${res.status}`)
  }
  return envelope.data as T
}

export function getCompanyProfile(token: string): Promise<CompanyProfile> {
  return send<CompanyProfile>('GET', token)
}

export function updateCompanyProfile(token: string, input: CompanyProfileInput): Promise<CompanyProfile> {
  return send<CompanyProfile>('PUT', token, input)
}
