/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_API_BASE_URL: string
  readonly VITE_LM2_LOGIN_URL?: string
  readonly VITE_LM2_CALLBACK_PATH?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
