/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_API_BASE_URL: string
  readonly VITE_LM2_LOGIN_URL?: string
  readonly VITE_LM2_CALLBACK_PATH?: string
  // Set only once storefront-frontend is actually deployed somewhere (see
  // docker-compose.yml's storefront service) -- absent means no live
  // storefront to link to yet.
  readonly VITE_STOREFRONT_URL?: string
  // "classic" pins the pre-2026-10 admin design for everyone who hasn't
  // picked one in the account menu (rollback switch); anything else means
  // the depot design. See theme/designPreference.
  readonly VITE_DESIGN_DEFAULT?: string
  // The deployed release (e.g. "1.0.11"): only ever from the server's
  // /config.js -- deploy.sh writes it to .env. Absent in local dev.
  readonly COMMERCEOS_VERSION?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}

// Written by the frontend container at startup (/config.js, from the
// server's .env) -- see src/config/appEnv.ts.
// eslint-disable-next-line no-var
declare var __COMMERCEOS_CONFIG__: Partial<Record<keyof ImportMetaEnv, string>> | undefined
