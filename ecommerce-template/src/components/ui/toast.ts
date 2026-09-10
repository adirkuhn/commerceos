import { create } from 'zustand'

export interface ToastMessage {
  id: string
  title: string
  description?: string
  variant?: 'default' | 'destructive' | 'success'
}

interface ToastStoreState {
  toasts: ToastMessage[]
  push: (toast: Omit<ToastMessage, 'id'>) => void
  dismiss: (id: string) => void
}

// Deliberately not built on Radix's Toast primitive -- a single visible
// toast at a time with plain CSS transitions is enough here, and it
// keeps the store (what to show) separate from Toaster.tsx (how to show
// it), so any component can call toast(...) without importing React.
export const useToastStore = create<ToastStoreState>((set) => ({
  toasts: [],
  push: (toast) => {
    const id = typeof crypto !== 'undefined' && crypto.randomUUID ? crypto.randomUUID() : `toast-${Date.now()}`
    set(s => ({ toasts: [...s.toasts, { ...toast, id }] }))
    setTimeout(() => set(s => ({ toasts: s.toasts.filter(t => t.id !== id) })), 4000)
  },
  dismiss: (id) => set(s => ({ toasts: s.toasts.filter(t => t.id !== id) })),
}))

export function toast(message: Omit<ToastMessage, 'id'>): void {
  useToastStore.getState().push(message)
}
