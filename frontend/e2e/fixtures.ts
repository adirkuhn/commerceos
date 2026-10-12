import { createHmac, randomUUID } from 'node:crypto'
import { test as base, expect, type APIRequestContext, type Page } from '@playwright/test'

// Shared E2E setup: a signed-in owner session for the real local backend.
//
// The backend's CORS only allows the deployed frontend origin, so API calls
// from the dev server are routed through Playwright and answered with a CORS
// header for this origin. Request and response bodies pass through untouched.
const SECRET = process.env.E2E_JWT_SECRET ?? ''
const TENANT_ID = process.env.E2E_TENANT_ID ?? ''
export const API_BASE_URL = process.env.E2E_API_BASE_URL ?? 'https://api-commerceos.dev.localhost/api/v1'
export const CONFIGURED = SECRET !== '' && TENANT_ID !== ''
export const NOT_CONFIGURED_REASON =
  'E2E needs E2E_JWT_SECRET (backend JWT_SECRET) and E2E_TENANT_ID (a tenant_settings.tenant_id) -- see playwright.config.ts'

function b64url(value: object): string {
  return Buffer.from(JSON.stringify(value)).toString('base64url')
}

// Same shape lm2 issues (backend/infrastructure/middleware/auth.go Claims).
export function mintToken(role = 'customer_owner'): string {
  const now = Math.floor(Date.now() / 1000)
  const header = b64url({ alg: 'HS256', typ: 'JWT' })
  const payload = b64url({ user_id: randomUUID(), tenant_id: TENANT_ID, role, iat: now, exp: now + 3600 })
  const signature = createHmac('sha256', SECRET).update(`${header}.${payload}`).digest('base64url')
  return `${header}.${payload}.${signature}`
}

async function signIn(page: Page, token: string, appearance: 'light' | 'dark' | 'system', design: 'depot' | 'classic') {
  await page.addInitScript(([jwt, theme, look]) => {
    localStorage.setItem('commerceos_jwt', jwt)
    localStorage.setItem('commerceos_scanner_jwt', jwt)
    // Initial appearance only -- a choice made during the test must survive reloads.
    if (localStorage.getItem('commerceos.theme') === null) localStorage.setItem('commerceos.theme', theme)
    if (localStorage.getItem('commerceos.design') === null) localStorage.setItem('commerceos.design', look)
  }, [token, appearance, design] as const)

  const apiOrigin = new URL(API_BASE_URL).origin
  await page.route(`${apiOrigin}/**`, async route => {
    const origin = await route.request().headerValue('origin')
    const cors = {
      'access-control-allow-origin': origin ?? '*',
      'access-control-allow-headers': 'Content-Type, Authorization',
      'access-control-allow-methods': 'GET, POST, PATCH, PUT, DELETE, OPTIONS',
    }
    if (route.request().method() === 'OPTIONS') {
      await route.fulfill({ status: 204, headers: cors })
      return
    }
    const response = await route.fetch()
    await route.fulfill({ response, headers: { ...response.headers(), ...cors } })
  })
}

interface Fixtures {
  token: string
  appearance: 'light' | 'dark' | 'system'
  design: 'depot' | 'classic'
}

export const test = base.extend<Fixtures>({
  appearance: ['light', { option: true }],
  design: ['depot', { option: true }],
  token: async ({}, use) => { await use(CONFIGURED ? mintToken() : '') },
  page: async ({ page, token, appearance, design }, use) => {
    test.skip(!CONFIGURED, NOT_CONFIGURED_REASON)
    await signIn(page, token, appearance, design)
    await use(page)
  },
})

export { expect }

// Direct API call with the test session (setup/teardown of test data).
// Uses Playwright's request context, which honours ignoreHTTPSErrors.
export async function api<T>(
  request: APIRequestContext, token: string, path: string, init: { method: string; body?: unknown },
): Promise<T> {
  const res = await request.fetch(`${API_BASE_URL}${path}`, {
    method: init.method,
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
    data: init.body,
  })
  const body = await res.json().catch(() => ({}))
  if (!res.ok()) throw new Error(`${init.method} ${path} -> ${res.status()} ${JSON.stringify(body)}`)
  return (body as { data: T }).data
}
