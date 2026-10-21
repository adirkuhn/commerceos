import type { ReactNode } from 'react'

export const inputClass = 'w-full px-2.5 py-1.5 text-xs rounded-lg bg-surface border border-line text-ink'
export const labelClass = 'block text-2xs font-medium text-ink-3 mb-1'
export const primaryButtonClass = 'px-4 py-2 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover disabled:opacity-50 transition-colors'

// One titled section of a carrier settings page (a labelled region).
export function SettingsSection({ id, title, body, children }: { id: string; title: string; body: string; children: ReactNode }) {
  return (
    <section aria-labelledby={`${id}-title`} className="space-y-3">
      <div>
        <h3 id={`${id}-title`} className="text-xs font-semibold text-ink-2">{title}</h3>
        <p className="text-2xs text-ink-4">{body}</p>
      </div>
      {children}
    </section>
  )
}
