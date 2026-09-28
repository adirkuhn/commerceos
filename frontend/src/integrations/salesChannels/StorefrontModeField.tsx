import type { StorefrontMode } from '../types'

const OPTIONS: { value: StorefrontMode; label: string; help: string }[] = [
  { value: 'headless', label: 'Headless (API only)', help: 'Your own frontend talks to the storefront API. CommerceOS hosts no shop pages.' },
  { value: 'theme', label: 'Hosted storefront', help: 'CommerceOS also hosts a shop for this channel, designed with the theme editor.' },
]

interface Props {
  value: StorefrontMode
  onChange: (mode: StorefrontMode) => void
}

export default function StorefrontModeField({ value, onChange }: Props) {
  const help = OPTIONS.find(o => o.value === value)?.help
  return (
    <div>
      <label htmlFor="channel-storefront-mode" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
        Storefront
      </label>
      <select
        id="channel-storefront-mode"
        value={value}
        onChange={e => onChange(e.target.value as StorefrontMode)}
        className="w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus"
      >
        {OPTIONS.map(o => <option key={o.value} value={o.value}>{o.label}</option>)}
      </select>
      {help && <p className="mt-1.5 text-2xs text-ink-4">{help}</p>}
    </div>
  )
}
