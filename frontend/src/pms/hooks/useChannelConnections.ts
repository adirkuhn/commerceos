import { useEffect, useState } from 'react'
import * as integrationsApi from '../../integrations/api'
import type { Connection } from '../../integrations/types'

export interface ChannelConnections {
  woo: Connection | null
  bol: Connection | null
}

// Integration tabs only ever appear for a provider that's actually
// connected — an unconnected/disabled integration gets no tab at all,
// rather than a tab that just tells you to go connect it. A failed lookup
// counts as "nothing connected".
export function useChannelConnections(token: string | null): ChannelConnections {
  const [woo, setWoo] = useState<Connection | null>(null)
  const [bol, setBol] = useState<Connection | null>(null)
  useEffect(() => {
    if (!token) return
    integrationsApi.listConnections(token)
      .then(conns => {
        setWoo(conns.find(c => c.provider === 'woocommerce' && c.status === 'connected') ?? null)
        setBol(conns.find(c => c.provider === 'bol' && c.status === 'connected') ?? null)
      })
      .catch(() => { setWoo(null); setBol(null) })
  }, [token])
  return { woo, bol }
}
