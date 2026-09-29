import i18next from 'i18next'
import type { Connection } from './types'

/** A connection's status in the admin language (lowercase; shown capitalised by CSS). */
export function connectionStatusLabel(status: Connection['status']): string {
  return status === 'connected' ? i18next.t('integrations:status.connected') : i18next.t('integrations:status.error')
}
