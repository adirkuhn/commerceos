import type { Customer } from './customer'

export interface AuthSession {
  customer: Customer
  token: string
  expiresAt: string
  twoFactorVerified: boolean
}

export interface LoginCredentials {
  email: string
  password: string
}

export interface RegisterInput {
  firstName: string
  lastName: string
  email: string
  password: string
}
