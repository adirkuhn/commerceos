// AuthService abstracts session management behind an interface so a real
// backend's own auth flow (JWT/OAuth/session cookie, whatever it ends up
// being) can replace MockAuthService without touching stores/authStore.ts
// or any component.
import type { AuthSession, Customer } from '@/types'
import { loginCustomer, registerCustomer } from '@/api/customers'

export interface AuthService {
  login(email: string, password: string): Promise<AuthSession>
  register(input: { firstName: string; lastName: string; email: string; password: string }): Promise<AuthSession>
  requestPasswordReset(email: string): Promise<void>
  resetPassword(token: string, newPassword: string): Promise<void>
  verifyTwoFactorCode(code: string): Promise<boolean>
}

function toSession(customer: Customer): AuthSession {
  return {
    customer,
    token: `mock-session-${customer.id}-${Date.now()}`,
    expiresAt: new Date(Date.now() + 1000 * 60 * 60 * 24 * 7).toISOString(),
    twoFactorVerified: true,
  }
}

class MockAuthService implements AuthService {
  async login(email: string, password: string): Promise<AuthSession> {
    const customer = await loginCustomer(email, password)
    return toSession(customer)
  }

  async register(input: { firstName: string; lastName: string; email: string; password: string }): Promise<AuthSession> {
    const customer = await registerCustomer(input)
    return toSession(customer)
  }

  async requestPasswordReset(_email: string): Promise<void> {
    // A real implementation emails a reset link. The mock just simulates
    // network latency via the caller's own loading state, nothing to do.
    return Promise.resolve()
  }

  async resetPassword(token: string, _newPassword: string): Promise<void> {
    if (!token) throw new Error('Reset link is invalid or has expired.')
    return Promise.resolve()
  }

  async verifyTwoFactorCode(code: string): Promise<boolean> {
    return Promise.resolve(code.trim().length === 6)
  }
}

export const authService: AuthService = new MockAuthService()
