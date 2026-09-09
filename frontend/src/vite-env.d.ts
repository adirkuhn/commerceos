/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_API_BASE_URL: string
  readonly VITE_LM2_LOGIN_URL?: string
  readonly VITE_LM2_CALLBACK_PATH?: string
  // Set only once storefront-frontend is actually deployed somewhere (see
  // docker-compose.yml's storefront service) -- absent means no live
  // storefront to link to yet.
  readonly VITE_STOREFRONT_URL?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
