import { CodedApiError, codedRequest } from '../codedRequest'
import type { CompanyProfile, CompanyProfileField, CompanyProfileInput } from './types'
import { appEnv } from '../../config/appEnv'

function url(): string {
  return `${appEnv().VITE_API_BASE_URL}/settings/company-profile`
}

// Backend error code -> the form field it belongs to (see
// backend/infrastructure/handlers/settings/company_profile_handler.go).
const FIELD_BY_CODE: Readonly<Record<string, CompanyProfileField>> = {
  COMPANY_LEGAL_NAME_REQUIRED: 'legal_name',
  COMPANY_INVALID_EMAIL: 'email',
  COMPANY_INVALID_VAT_NUMBER: 'vat_number',
  COMPANY_INVALID_IBAN: 'iban',
  COMPANY_INVALID_COUNTRY: 'address.country',
}

export class CompanyProfileError extends CodedApiError {
  readonly field: CompanyProfileField | undefined

  constructor(code: string, message: string) {
    super(code, message)
    this.name = 'CompanyProfileError'
    this.field = FIELD_BY_CODE[code]
  }
}

async function send<T>(method: 'GET' | 'PUT', token: string, body?: unknown): Promise<T> {
  try {
    return await codedRequest<T>(method, url(), token, body)
  } catch (e) {
    if (e instanceof CodedApiError) throw new CompanyProfileError(e.code, e.message)
    throw e
  }
}

export function getCompanyProfile(token: string): Promise<CompanyProfile> {
  return send<CompanyProfile>('GET', token)
}

export function updateCompanyProfile(token: string, input: CompanyProfileInput): Promise<CompanyProfile> {
  return send<CompanyProfile>('PUT', token, input)
}
