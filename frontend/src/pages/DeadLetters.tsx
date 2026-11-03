import { Inbox, ShieldAlert } from 'lucide-react'
import { useLocation } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useAuth } from '../auth/AuthContext'
import { subPathSegments } from '../routes'
import { DeadLetterDetail } from '../deadletters/DeadLetterDetail'
import { DeadLetterList } from '../deadletters/DeadLetterList'
import { DEAD_LETTERS_PATH, parseDeadLetterRef } from '../deadletters/format'

// DeadLetters is the super_admin dead-letter admin (COM-14): the list at
// /dead-letters, one dead letter at /dead-letters/<stream>/<seq>. The API
// is gated identically (RequireRole(super_admin)).
export default function DeadLetters() {
  const { t } = useTranslation('settings')
  const { user } = useAuth()
  const { pathname } = useLocation()

  if (user?.role !== 'super_admin') {
    return (
      <div className="flex flex-col items-center justify-center h-full gap-2 text-ink-4">
        <ShieldAlert size={24} />
        <p className="text-sm">{t('jobs.superAdminOnly')}</p>
      </div>
    )
  }

  const ref = parseDeadLetterRef(subPathSegments(pathname, DEAD_LETTERS_PATH))
  return (
    <div className="flex flex-col h-full overflow-auto">
      <div className="flex items-center gap-2 px-5 py-3 border-b border-line bg-sunken shrink-0">
        <Inbox size={14} className="text-accent" />
        <h1 className="text-sm font-semibold text-ink">{t('deadLetters.title')}</h1>
      </div>
      {ref ? <DeadLetterDetail key={`${ref.stream}/${ref.seq}`} stream={ref.stream} seq={ref.seq} /> : <DeadLetterList />}
    </div>
  )
}
