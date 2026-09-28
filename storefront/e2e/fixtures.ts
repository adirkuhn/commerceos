import { createHmac, randomUUID } from 'node:crypto'
import type { Page } from '@playwright/test'

const SECRET = process.env.E2E_JWT_SECRET ?? ''
const TENANT_ID = process.env.E2E_TENANT_ID ?? ''
export const API_BASE_URL = process.env.E2E_API_BASE_URL ?? 'https://api-commerceos.dev.localhost/api/v1'
export const CONFIGURED = SECRET !== '' && TENANT_ID !== ''
export const NOT_CONFIGURED_REASON = 'E2E needs E2E_JWT_SECRET and E2E_TENANT_ID -- see playwright.config.ts'

const b64url = (v: object) => Buffer.from(JSON.stringify(v)).toString('base64url')

/** A merchant JWT like lm2 issues (backend middleware.Claims). */
export function ownerToken(): string {
  const now = Math.floor(Date.now() / 1000)
  const header = b64url({ alg: 'HS256', typ: 'JWT' })
  const payload = b64url({ user_id: randomUUID(), tenant_id: TENANT_ID, role: 'customer_owner', iat: now, exp: now + 3600 })
  return `${header}.${payload}.${createHmac('sha256', SECRET).update(`${header}.${payload}`).digest('base64url')}`
}

/** The backend's CORS only allows the deployed admin origin; answer for this one. */
export async function allowCrossOriginAPI(page: Page) {
  const apiOrigin = new URL(API_BASE_URL).origin
  await page.route(`${apiOrigin}/**`, async route => {
    const cors = {
      'access-control-allow-origin': (await route.request().headerValue('origin')) ?? '*',
      'access-control-allow-headers': 'Content-Type, Authorization, X-Theme-Preview',
      'access-control-allow-methods': 'GET, POST, PATCH, PUT, DELETE, OPTIONS',
    }
    if (route.request().method() === 'OPTIONS') return route.fulfill({ status: 204, headers: cors })
    const response = await route.fetch()
    return route.fulfill({ response, headers: { ...response.headers(), ...cors } })
  })
}
