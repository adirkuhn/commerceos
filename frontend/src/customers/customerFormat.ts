import type { Customer } from './types'

export function fullName(c: Customer): string {
  return [c.first_name, c.last_name].filter(Boolean).join(' ') || c.email
}
