import clsx from 'clsx'
import { Loader2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import type { AssistantRequestStatus } from './types'

const STYLES: Record<AssistantRequestStatus, string> = {
  queued: 'bg-fill text-ink-3',
  processing: 'bg-accent/15 text-accent',
  needs_input: 'bg-amber-500/15 text-warning',
  answered: 'bg-green-500/15 text-success',
  cancelled: 'bg-fill text-ink-4',
}

export function StatusChip({ status }: { status: AssistantRequestStatus }) {
  const { t } = useTranslation('assistant')
  return (
    <span className={clsx('inline-flex items-center gap-1 px-1.5 py-0.5 rounded-full text-2xs font-medium whitespace-nowrap', STYLES[status])}>
      {status === 'processing' && <Loader2 size={10} className="animate-spin" aria-hidden />}
      {t(`status.${status}`)}
    </span>
  )
}
