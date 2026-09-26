import type { QueueLock } from '../oms/types'
import { isTenantAdmin } from '../auth/roles'

// Lock takeover is for tenant admins (infrastructure/handlers/oms
// lockActorFrom) -- only decides whether to offer the button; the backend
// enforces.
export function canTakeOverLocks(role: string | undefined): boolean {
  return isTenantAdmin(role)
}

export function formatLockAge(sinceIso: string, now: number = Date.now()): string {
  const minutes = Math.max(0, Math.floor((now - new Date(sinceIso).getTime()) / 60_000))
  if (minutes < 1) return '<1 min'
  if (minutes < 60) return `${minutes} min`
  const hours = Math.floor(minutes / 60)
  const rest = minutes % 60
  return rest === 0 ? `${hours} h` : `${hours} h ${rest} min`
}

export function shortUserId(userId: string): string {
  return `User ${userId.slice(0, 6)}`
}

const ACTIVITY_LABEL: Record<QueueLock['activity'], string> = { picking: 'Picking', packing: 'Packing' }

export function lockLabel(lock: QueueLock, name: string, now: number = Date.now()): string {
  return `${ACTIVITY_LABEL[lock.activity]} · ${name} · ${formatLockAge(lock.since, now)}`
}
