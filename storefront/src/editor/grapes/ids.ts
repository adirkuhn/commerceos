/** A short, URL/CSS-safe node id (matches the contract's id pattern). */
export function newNodeId(): string {
  const bytes = crypto.getRandomValues(new Uint8Array(6))
  return 'n' + Array.from(bytes, b => b.toString(36).padStart(2, '0')).join('').slice(0, 10)
}
