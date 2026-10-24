import { useEffect, useState } from 'react'
import { fromMinorUnits, toMinorUnits } from '../../collections/form'

// Money typed in major units ("12,50"), committed as minor units on blur
// or Enter; an empty or invalid entry goes back to the last value.
export function MoneyField({ value, onChange, label, className }: { value: number; onChange: (minor: number) => void; label: string; className?: string }) {
  const [text, setText] = useState(fromMinorUnits(value))
  useEffect(() => { setText(fromMinorUnits(value)) }, [value])

  function commit() {
    const minor = text.trim() === '' ? 0 : toMinorUnits(text)
    if (minor === null) setText(fromMinorUnits(value))
    else if (minor !== value) onChange(minor)
  }

  return (
    <input
      value={text} inputMode="decimal" aria-label={label}
      onChange={e => setText(e.target.value)} onBlur={commit}
      onKeyDown={e => { if (e.key === 'Enter') { e.preventDefault(); e.currentTarget.blur() } }}
      className={className ?? 'w-24 px-2 py-1.5 text-xs text-right tabular-nums rounded-lg bg-raised border border-line text-ink outline-none focus:ring-1 focus:ring-focus'}
    />
  )
}
