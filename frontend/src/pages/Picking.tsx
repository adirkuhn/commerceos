
import { useAuth } from '../auth/AuthContext'
import { usePickingMode } from '../picking/usePickingMode'
import { PickingListMode } from '../picking/PickingListMode'
import { ScanPicking } from '../picking/ScanPicking'

// Picking follows the tenant's picking mode (Settings > Fulfillment): list
// mode is just "see the items, mark the order picked"; scan mode is the
// full scan-into-a-session flow below.
export default function Picking() {
  const { token } = useAuth()
  const { mode, error } = usePickingMode(token ?? '')
  if (!token || mode === null) {
    return <div className="flex h-full items-center justify-center text-sm text-ink-4">Loading…</div>
  }
  return (
    <div className="flex flex-col h-full">
      {error && <p role="alert" className="px-4 py-2 text-2xs text-danger border-b border-line">Couldn&apos;t load the picking mode ({error}) -- showing scan mode.</p>}
      <div className="flex-1 min-h-0">
        {mode === 'list' ? <PickingListMode token={token} /> : <ScanPicking />}
      </div>
    </div>
  )
}
