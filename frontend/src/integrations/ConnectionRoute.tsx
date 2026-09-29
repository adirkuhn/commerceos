import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuth } from '../auth/AuthContext'
import { useConnections } from './hooks/useConnections'
import { RecordLoading, RecordNotFound } from '../components/common/RecordStatus'
import type { Connection } from './types'

interface ConnectionRouteProps {
  connectionId: string
  onBack: () => void
  children: (connection: Connection) => ReactNode
}

// Resolves a connection id taken from the URL (e.g.
// /settings/integrations/<id>) to the Connection a settings page needs, so
// that page survives a refresh instead of depending on the object having
// been passed along in memory from a click.
export function ConnectionRoute({ connectionId, onBack, children }: ConnectionRouteProps) {
  const { token } = useAuth()
  const { t } = useTranslation('integrations')
  const { connections, loading } = useConnections(token ?? '')
  const connection = connections.find(c => c.id === connectionId)
  if (connection) return <>{children(connection)}</>
  return loading ? <RecordLoading /> : <RecordNotFound label={t('connection')} onBack={onBack} />
}
