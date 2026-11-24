import { useId } from 'react'
import clsx from 'clsx'

export interface FilterChipOption<T extends string> {
  value: T
  label: string
}

interface FilterChipsProps<T extends string> {
  label: string
  options: readonly FilterChipOption<T>[]
  selected: readonly T[]
  onToggle: (value: T) => void
  className?: string
}

// A multi-select filter as joined chips under a tab-caps label, like the
// ruled boxes on a fare coupon. Each chip is a toggle button (aria-pressed).
export function FilterChips<T extends string>({ label, options, selected, onToggle, className }: FilterChipsProps<T>) {
  const labelId = useId()
  return (
    <div role="group" aria-labelledby={labelId} className={clsx('flex flex-col gap-1', className)}>
      <span id={labelId} className="caps-label-widest text-2xs font-semibold text-ink-4">{label}</span>
      <div className="flex flex-wrap rounded-md border border-line bg-surface overflow-hidden divide-x divide-line">
        {options.map(({ value, label: optionLabel }) => {
          const on = selected.includes(value)
          return (
            <button
              key={value}
              type="button"
              onClick={() => onToggle(value)}
              aria-pressed={on}
              className={clsx(
                'h-7 px-2.5 text-xs font-semibold whitespace-nowrap transition-colors duration-150 focus-visible:relative focus-visible:z-10',
                on ? 'bg-accent-solid text-white' : 'text-ink-3 hover:bg-hover hover:text-ink',
              )}
            >
              {optionLabel}
            </button>
          )
        })}
      </div>
    </div>
  )
}
