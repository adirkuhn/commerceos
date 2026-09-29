import i18next from 'i18next'
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
  if (minutes < 1) return i18next.t('packing:lock.lessThanMinute')
  if (minutes < 60) return i18next.t('packing:lock.minutes', { minutes })
  const hours = Math.floor(minutes / 60)
  const rest = minutes % 60
  return rest === 0 ? i18next.t('packing:lock.hours', { hours }) : i18next.t('packing:lock.hoursMinutes', { hours, minutes: rest })
}

export function shortUserId(userId: string): string {
  return i18next.t('packing:lock.user', { id: userId.slice(0, 6) })
}

function activityLabel(activity: QueueLock['activity']): string {
  return activity === 'picking' ? i18next.t('packing:lock.activity.picking') : i18next.t('packing:lock.activity.packing')
}

export function lockLabel(lock: QueueLock, name: string, now: number = Date.now()): string {
  return `${activityLabel(lock.activity)} · ${name} · ${formatLockAge(lock.since, now)}`
}
