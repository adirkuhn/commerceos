import { useId } from 'react'
import clsx from 'clsx'
import type { Look } from '../../theme/lookPreference'

// Small form controls for the appearance section.

export const SECTION_LABEL = 'caps-label-widest text-2xs font-bold text-ink-3 mb-2'

interface SegmentedProps<T extends string> {
  name: string
  legend: string
  value: T
  options: ReadonlyArray<{ value: T; label: string }>
  disabled: boolean
  onChange: (value: T) => void
}

// Segmented is a radio group drawn as one row of buttons.
export function Segmented<T extends string>({ name, legend, value, options, disabled, onChange }: SegmentedProps<T>) {
  return (
    <fieldset disabled={disabled}>
      <legend className={SECTION_LABEL}>{legend}</legend>
      <div className="inline-flex p-0.5 rounded-md bg-fill">
        {options.map(o => (
          <label
            key={o.value}
            className={clsx(
              'px-3 py-1.5 rounded-md text-sm cursor-pointer has-[:focus-visible]:ring-2 has-[:focus-visible]:ring-focus/60',
              value === o.value ? 'bg-raised text-ink font-medium shadow-sm' : 'text-ink-3 hover:text-ink',
              disabled && 'cursor-default',
            )}
          >
            <input type="radio" name={name} value={o.value} checked={value === o.value} onChange={() => onChange(o.value)} className="sr-only" />
            {o.label}
          </label>
        ))}
      </div>
    </fieldset>
  )
}

// Each look's own colours (index.css), shown as a strip on its card.
export const LOOK_SWATCHES: Readonly<Record<Look, readonly string[]>> = {
  wallet: ['#14284B', '#EEF0F3', '#FFFFFF', '#4B2E83', '#C42B2B'],
  graphite: ['#1B1D21', '#25282D', '#E6E6E3', '#7FB4FF', '#F07A6A'],
}

interface LookCardsProps {
  label: string
  value: Look
  options: ReadonlyArray<{ value: Look; name: string; description: string }>
  disabled: boolean
  onChange: (value: Look) => void
}

// LookCards picks a look from cards showing its name, a line about it and
// a strip of its colours.
export function LookCards({ label, value, options, disabled, onChange }: LookCardsProps) {
  const id = useId()
  return (
    <div role="radiogroup" aria-labelledby={`${id}-label`}>
      <p id={`${id}-label`} className={SECTION_LABEL}>{label}</p>
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
        {options.map(o => {
          const checked = value === o.value
          return (
            <label
              key={o.value}
              className={clsx(
                'flex items-start gap-3 rounded-md border px-4 py-3 has-[:focus-visible]:ring-2 has-[:focus-visible]:ring-focus/60',
                checked ? 'border-accent bg-accent/5' : 'border-line bg-raised',
                disabled ? 'cursor-default' : 'cursor-pointer hover:border-ink-4',
              )}
            >
              <input
                type="radio"
                name={`${id}-look`}
                value={o.value}
                checked={checked}
                disabled={disabled}
                onChange={() => onChange(o.value)}
                aria-labelledby={`${id}-${o.value}-name`}
                aria-describedby={`${id}-${o.value}-description`}
                className="mt-1 accent-[rgb(var(--accent))]"
              />
              <span className="min-w-0 flex-1">
                <span id={`${id}-${o.value}-name`} className="block text-sm font-semibold text-ink">{o.name}</span>
                <span id={`${id}-${o.value}-description`} className="block text-xs text-ink-3">{o.description}</span>
                <span className="mt-2 flex h-3 overflow-hidden rounded-sm border border-line" aria-hidden>
                  {LOOK_SWATCHES[o.value].map(color => <span key={color} className="flex-1" style={{ background: color }} />)}
                </span>
              </span>
            </label>
          )
        })}
      </div>
    </div>
  )
}
