// The admin app opens the editor at /store/editor#session=<token>. The
// fragment never reaches a server (or its logs); we move it into
// sessionStorage (this tab only) and strip it from the address bar.

const KEY = 'cos-editor-session'

interface LocationLike {
  hash: string
  pathname: string
  search: string
}

export function takeSessionToken(
  location: LocationLike, storage: Storage, replaceURL: (url: string) => void,
): string | undefined {
  const fromHash = new URLSearchParams(location.hash.replace(/^#/, '')).get('session')
  if (fromHash) {
    storage.setItem(KEY, fromHash)
    replaceURL(location.pathname + location.search)
    return fromHash
  }
  return storage.getItem(KEY) ?? undefined
}
