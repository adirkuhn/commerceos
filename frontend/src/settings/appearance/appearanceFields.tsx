import type { ReactNode } from 'react'
import clsx from 'clsx'

// Small form controls for the appearance section.

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
      <legend className="text-sm font-medium text-ink mb-2">{legend}</legend>
      <div className="inline-flex p-0.5 rounded-lg bg-fill">
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

interface ColorFieldProps {
  label: string
  value: string
  disabled: boolean
  onChange: (value: string) => void
  children?: ReactNode
}

const HEX = /^#[0-9a-fA-F]{6}$/

export function isHexColor(value: string): boolean {
  return HEX.test(value)
}

// ColorField pairs the browser's colour picker with an editable hex value.
export function ColorField({ label, value, disabled, onChange, children }: ColorFieldProps) {
  return (
    <div>
      <p className="text-sm font-medium text-ink mb-2">{label}</p>
      <div className="flex items-center gap-2">
        <input
          type="color"
          aria-label={label}
          value={isHexColor(value) ? value : '#000000'}
          disabled={disabled}
          onChange={e => onChange(e.target.value.toUpperCase())}
          className="h-9 w-12 rounded-lg border border-line bg-surface p-1 cursor-pointer disabled:cursor-default"
        />
        <input
          aria-label={`${label} hex`}
          value={value}
          readOnly={disabled}
          maxLength={7}
          spellCheck={false}
          onChange={e => onChange(e.target.value.trim().toUpperCase())}
          aria-invalid={!isHexColor(value)}
          className="w-28 px-3 py-2 rounded-lg bg-surface border border-line font-mono text-sm text-ink uppercase focus:outline-none focus:ring-2 focus:ring-focus/60 aria-[invalid=true]:border-danger read-only:bg-sunken"
        />
      </div>
      {children}
    </div>
  )
}
