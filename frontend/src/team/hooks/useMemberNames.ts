import { useCallback, useEffect, useState } from 'react'
import * as api from '../api'
import { shortUserId } from '../../packing/lockDisplay'

// Resolves lm2 user ids to display names via lm2's member list. Best
// effort by design: lm2 may refuse the list to non-owner or scanner tokens,
// and a name is only a nicety next to a lock -- unknown ids fall back to a
// short id instead of surfacing an error.
export function useMemberNames(token: string): (userId: string) => string {
  const [names, setNames] = useState<ReadonlyMap<string, string>>(new Map())

  useEffect(() => {
    if (!token) return
    let cancelled = false
    api.listMembers(token)
      .then(members => {
        if (cancelled) return
        setNames(new Map(members.map(m => [m.user_id, [m.first_name, m.last_name].filter(Boolean).join(' ') || m.email])))
      })
      .catch(err => { console.warn('useMemberNames: lm2 member lookup failed, showing short ids', err) })
    return () => { cancelled = true }
  }, [token])

  return useCallback((userId: string) => names.get(userId) ?? shortUserId(userId), [names])
}
