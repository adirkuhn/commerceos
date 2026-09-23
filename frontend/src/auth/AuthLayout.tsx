import type { ReactNode } from 'react'
import { BrandMark } from '../components/brand/BrandMark'

// Shared frame for the pre-sign-in screens (login, OAuth callback, invite):
// a centred card on the app canvas, themed with the same tokens as the app.
export function AuthLayout({ title, subtitle, children }: { title: string; subtitle?: ReactNode; children?: ReactNode }) {
  return (
    <main className="min-h-screen flex items-center justify-center bg-canvas px-4">
      <div className="w-full max-w-sm rounded-2xl bg-raised border border-line shadow-xl p-8">
        <div className="w-10 h-10 mb-5 rounded-xl bg-accent-solid flex items-center justify-center">
          <BrandMark size={24} className="text-white" />
        </div>
        <h1 className="text-xl font-semibold text-ink">{title}</h1>
        {subtitle && <p className="mt-1.5 text-sm text-ink-3">{subtitle}</p>}
        {children && <div className="mt-6">{children}</div>}
      </div>
    </main>
  )
}

export const primaryButtonClass =
  'w-full py-2.5 rounded-lg bg-accent-solid text-white text-sm font-medium hover:bg-accent-solid-hover disabled:opacity-60 transition-colors'
export const secondaryButtonClass =
  'w-full py-2.5 rounded-lg bg-surface border border-line text-ink text-sm font-medium hover:bg-hover disabled:opacity-60 transition-colors'
export const inputClass =
  'w-full px-3 py-2 rounded-lg bg-surface border border-line text-sm text-ink placeholder-ink-4 focus:outline-none focus:ring-2 focus:ring-focus/60'
export const labelClass = 'block mb-1 text-xs font-medium text-ink-3'
