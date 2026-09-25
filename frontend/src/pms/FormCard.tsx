import clsx from 'clsx'

export function FormCard({ title, description, children }: { title: string; description?: string; children: React.ReactNode }) {
  return (
    <div className="rounded-xl border border-line p-4 space-y-3">
      <div>
        <h3 className="text-xs font-semibold text-ink">{title}</h3>
        {description && <p className="text-2xs text-ink-4 mt-0.5">{description}</p>}
      </div>
      {children}
    </div>
  )
}

export function ToggleRow({ label, checked, onChange }: { label: string; checked: boolean; onChange: () => void }) {
  return (
    <div className="flex items-center justify-between">
      <span className="text-xs font-medium text-ink">{label}</span>
      <button
        type="button"
        role="switch"
        aria-checked={checked}
        aria-label={label}
        onClick={onChange}
        className={clsx(
          'relative w-9 h-5 rounded-full shrink-0 transition-colors',
          checked ? 'bg-accent-solid' : 'dark:bg-night-700 bg-gray-300',
        )}
      >
        <span className={clsx(
          'absolute top-0.5 left-0.5 w-4 h-4 rounded-full bg-white transition-transform',
          checked ? 'translate-x-4' : 'translate-x-0',
        )} />
      </button>
    </div>
  )
}
