// parseEmailList turns free text (one address per line, or comma/semicolon
// separated) into a list of trimmed addresses. Validation is the backend's.
export function parseEmailList(text: string): string[] {
  return text.split(/[\n,;]/).map(s => s.trim()).filter(Boolean)
}
