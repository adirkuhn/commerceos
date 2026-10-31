// The admin's settings (API URL, lm2 login, storefront, default design).
//
// In production they come from /config.js, which the frontend container
// writes at startup from the server's .env (see frontend/docker/) -- so one
// image serves any server, and changing a URL is a restart, not a rebuild.
// Anything it leaves blank or doesn't set falls back to the build-time
// import.meta.env value, which is what `npm run dev` and the tests use.
//
// Read at call time, never cached: config.js loads before the app, and tests
// stub the environment per test.

export type AppEnv = ImportMetaEnv

const KEYS = [
  'VITE_API_BASE_URL',
  'VITE_LM2_LOGIN_URL',
  'VITE_LM2_CALLBACK_PATH',
  'VITE_STOREFRONT_URL',
  'VITE_DESIGN_DEFAULT',
] as const satisfies readonly (keyof ImportMetaEnv)[]

export function appEnv(): AppEnv {
  const runtime = globalThis.__COMMERCEOS_CONFIG__
  const env: Record<string, string | undefined> = {}
  for (const key of KEYS) {
    const value = runtime?.[key]
    env[key] = value ? value : (import.meta.env[key] as string | undefined)
  }
  return env as unknown as AppEnv
}
