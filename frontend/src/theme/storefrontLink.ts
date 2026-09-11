// storefrontOrigin is where storefront-frontend is actually reachable.
// VITE_STOREFRONT_URL overrides this once it's deployed on a dedicated
// domain of its own; until then it defaults to VITE_API_BASE_URL's own
// origin, since that's where it's path-mounted by default (see
// docker-compose.yml's storefront service) -- a real, working link with
// no extra configuration step. "" only if that default can't be computed
// (VITE_API_BASE_URL isn't a valid URL).
export function storefrontOrigin(): string {
  const explicit = import.meta.env.VITE_STOREFRONT_URL
  if (explicit) return explicit
  try {
    return new URL(import.meta.env.VITE_API_BASE_URL).origin
  } catch {
    return ''
  }
}

// STOREFRONT_PATH_PREFIX is where Traefik path-mounts storefront-frontend
// on storefrontOrigin()'s own domain (PathPrefix(`/store`) + stripprefix,
// see docker-compose.yml's storefront service) -- every link into it
// needs this prefix, storefrontOrigin()'s bare origin does not route there.
const STOREFRONT_PATH_PREFIX = '/store'

// storefrontEnterPath is every internal channel's own storefront entry
// point -- the channel's own id doubles as its storefront bearer key (see
// backend application/apikeys.ResolveAPIKeyUseCase's fallback). next, when
// given, is where storefront-frontend's StoreEntry sends the browser after
// remembering the key (defaults to "/", the home page) -- used to preview
// a specific page instead of always landing on home.
export function storefrontEnterPath(channelId: string, next?: string): string {
  const path = `${STOREFRONT_PATH_PREFIX}/enter/${channelId}`
  return next ? `${path}?next=${encodeURIComponent(next)}` : path
}

// storefrontVisitURL is the full clickable "Visit store" link -- an
// internal channel's one public entry point. Independent of any
// particular rendered frontend: whatever's deployed at storefrontOrigin()
// (today's storefront-frontend, or a future headless React storefront
// attached to this same channel id as its bearer key) is reached the same
// way.
export function storefrontVisitURL(channelId: string): string {
  const origin = storefrontOrigin()
  return origin ? `${origin}${storefrontEnterPath(channelId)}` : ''
}
