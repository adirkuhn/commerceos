// Per-store browser storage for shopper tokens (cart, session, wishlist).
// Blocked storage (private mode) degrades to "not remembered".

export function readToken(kind: string, storeKey: string): string | null {
  try {
    return window.localStorage.getItem(`cos-${kind}:${storeKey}`)
  } catch {
    return null
  }
}

export function writeToken(kind: string, storeKey: string, token: string | null) {
  try {
    if (token) window.localStorage.setItem(`cos-${kind}:${storeKey}`, token)
    else window.localStorage.removeItem(`cos-${kind}:${storeKey}`)
  } catch {
    // not remembered
  }
}
