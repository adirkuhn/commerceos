import type { StorefrontMode } from '../types'

const OPTIONS: { value: StorefrontMode; label: string; help: string }[] = [
  { value: 'headless', label: 'Headless (API only)', help: 'Your own frontend talks to the storefront API. CommerceOS hosts no shop pages.' },
  { value: 'theme', label: 'Hosted storefront', help: 'CommerceOS also hosts a shop for this channel, designed with the theme editor.' },
]

interface Props {
  value: StorefrontMode
  onChange: (mode: StorefrontMode) => void
  signInUrl: string
  onSignInUrlChange: (url: string) => void
}

const inputClass = 'w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus'
const labelClass = 'block text-2xs text-ink-4 uppercase tracking-wider mb-1.5'

export default function StorefrontModeField({ value, onChange, signInUrl, onSignInUrlChange }: Props) {
  const help = OPTIONS.find(o => o.value === value)?.help
  return (
    <>
      <div>
        <label htmlFor="channel-storefront-mode" className={labelClass}>Storefront</label>
        <select id="channel-storefront-mode" value={value} onChange={e => onChange(e.target.value as StorefrontMode)} className={inputClass}>
          {OPTIONS.map(o => <option key={o.value} value={o.value}>{o.label}</option>)}
        </select>
        {help && <p className="mt-1.5 text-2xs text-ink-4">{help}</p>}
      </div>
      {value === 'headless' && (
        <div>
          <label htmlFor="channel-sign-in-url" className={labelClass}>Customer sign-in page</label>
          <input id="channel-sign-in-url" type="url" inputMode="url" value={signInUrl} maxLength={2000}
            placeholder="https://shop.example.com/account/verify" onChange={e => onSignInUrlChange(e.target.value)} className={inputClass} />
          <p className="mt-1.5 text-2xs text-ink-4">
            The page on your own site that completes sign-in. Sign-in emails link here with ?token=… added. Leave empty to turn customer sign-in off.
          </p>
        </div>
      )}
    </>
  )
}
