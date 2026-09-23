import type { PackingQueueLock } from '../oms/types'

// Mirrors the backend's admin roles for lock takeover
// (infrastructure/handlers/oms lockActorFrom) -- only decides whether to
// offer the button; the backend enforces.
const LOCK_ADMIN_ROLES = new Set(['super_admin', 'customer_owner', 'customer_admin'])

export function canTakeOverLocks(role: string | undefined): boolean {
  return role !== undefined && LOCK_ADMIN_ROLES.has(role)
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

const ACTIVITY_LABEL: Record<PackingQueueLock['activity'], string> = { picking: 'Picking', packing: 'Packing' }

export function lockLabel(lock: PackingQueueLock, name: string, now: number = Date.now()): string {
  return `${ACTIVITY_LABEL[lock.activity]} · ${name} · ${formatLockAge(lock.since, now)}`
}
