import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { AuthSession, Customer } from '@/types'
import { authService } from '@/services/authService'
import { analytics } from '@/services/analytics'

interface AuthState {
  session: AuthSession | null
  isAuthenticated: boolean
  login: (email: string, password: string) => Promise<void>
  register: (input: { firstName: string; lastName: string; email: string; password: string }) => Promise<void>
  logout: () => void
  setCustomer: (customer: Customer) => void
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      session: null,
      isAuthenticated: false,
      login: async (email, password) => {
        const session = await authService.login(email, password)
        set({ session, isAuthenticated: true })
        analytics.track('login', { method: 'password' })
      },
      register: async (input) => {
        const session = await authService.register(input)
        set({ session, isAuthenticated: true })
        analytics.track('register', { method: 'password' })
      },
      logout: () => set({ session: null, isAuthenticated: false }),
      setCustomer: (customer) => {
        const session = get().session
        if (session) set({ session: { ...session, customer } })
      },
    }),
    { name: 'aurora-auth' },
  ),
)
