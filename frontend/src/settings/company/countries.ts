// Countries offered in the company address picker: EU/EEA, the UK,
// Switzerland and main trading partners. Names come from the browser
// (Intl.DisplayNames), so there is no list of names to maintain.
const CODES = [
  'AT', 'BE', 'BG', 'BR', 'CH', 'CY', 'CZ', 'DE', 'DK', 'EE', 'ES', 'FI', 'FR', 'GB', 'GR', 'HR', 'HU',
  'IE', 'IS', 'IT', 'LI', 'LT', 'LU', 'LV', 'MT', 'NL', 'NO', 'PL', 'PT', 'RO', 'SE', 'SI', 'SK', 'US',
] as const

function nameOf(code: string): string {
  try {
    return new Intl.DisplayNames(['en'], { type: 'region' }).of(code) ?? code
  } catch {
    return code
  }
}

export interface CountryOption {
  code: string
  name: string
}

// Sorted by name. A saved code outside the list is kept as an option so
// loading a profile never silently changes its country.
export function countryOptions(current = ''): CountryOption[] {
  const codes: string[] = [...CODES]
  if (current && !codes.includes(current)) codes.push(current)
  return codes.map(code => ({ code, name: nameOf(code) })).sort((a, b) => a.name.localeCompare(b.name))
}
