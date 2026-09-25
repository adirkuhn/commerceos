import type { CompanyMember } from '../team/types'

export function fullName(m: CompanyMember): string {
  return [m.first_name, m.last_name].filter(Boolean).join(' ') || m.email
}
