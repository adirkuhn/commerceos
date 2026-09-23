import { useState } from 'react'
import { useAuth } from '../auth/AuthContext'
import { useTenantSettings } from './hooks/useTenantSettings'
import { toInput } from './tenantSettingsInput'
import type { PickingMode } from './types'

const OPTIONS: { mode: PickingMode; label: string; description: string }[] = [
  {
    mode: 'scan', label: 'Scan each item',
    description: 'Pickers scan every item into a picking session before the order goes to packing.',
  },
  {
    mode: 'list', label: 'List',
    description: 'Pickers see each order’s items as a list and mark the whole order picked with one button -- no scanning. Items are scanned during packing.',
  },
]

// Settings > Fulfillment: how the picking module works for this tenant.
// Picking is optional either way -- packing is where items are scanned and
// labels made. Saves on change (a two-option choice needs no Save button).
export function PickingModeSection() {
  const { token } = useAuth()
  const { settings, save } = useTenantSettings(token ?? '')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [saved, setSaved] = useState(false)
  const current: PickingMode = settings?.picking_mode ?? 'scan'

  async function choose(mode: PickingMode) {
    if (mode === current || !settings) return
    setError('')
    setSaved(false)
    setSaving(true)
    try {
      await save({ ...toInput(settings), picking_mode: mode })
      setSaved(true)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save picking mode')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="bg-raised rounded-xl border border-line p-4 space-y-3">
      <div>
        <h3 className="text-sm font-semibold text-ink">Picking</h3>
        <p className="text-2xs text-ink-4 mt-0.5">How pickers work through orders before packing.</p>
      </div>
      <fieldset className="space-y-2" disabled={!settings || saving}>
        <legend className="sr-only">Picking mode</legend>
        {OPTIONS.map(o => (
          <label key={o.mode} className="flex items-start gap-2.5 p-3 rounded-lg border border-line bg-surface cursor-pointer has-[:checked]:border-accent">
            <input
              type="radio" name="picking-mode" value={o.mode}
              checked={current === o.mode}
              onChange={() => choose(o.mode)}
              className="mt-0.5 accent-accent"
            />
            <span>
              <span className="block text-xs font-medium text-ink">{o.label}</span>
              <span className="block text-2xs text-ink-3 mt-0.5">{o.description}</span>
            </span>
          </label>
        ))}
      </fieldset>
      {error && <p role="alert" className="text-2xs text-danger">{error}</p>}
      {saved && <p className="text-2xs text-success">Saved</p>}
    </div>
  )
}
