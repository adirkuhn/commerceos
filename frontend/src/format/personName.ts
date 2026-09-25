// NamedPerson is anything with a first/last name and an email -- customers,
// team members, invitees.
export interface NamedPerson {
  first_name?: string
  last_name?: string
  email: string
}

// fullName is "First Last" (whichever parts exist), else the email.
export function fullName(p: NamedPerson): string {
  return [p.first_name, p.last_name].filter(Boolean).join(' ') || p.email
}
