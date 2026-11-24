import clsx from 'clsx'
import { Loader2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import type { AssistantRequestStatus } from './types'

const STYLES: Record<AssistantRequestStatus, string> = {
  queued: 'bg-fill text-ink-3',
  processing: 'bg-accent/15 text-accent',
  needs_input: 'bg-warning/10 text-warning',
  answered: 'bg-success/10 text-success',
  awaiting_confirmation: 'bg-warning/10 text-warning',
  cancelled: 'bg-fill text-ink-4',
}

export function StatusChip({ status }: { status: AssistantRequestStatus }) {
  const { t } = useTranslation('assistant')
  return (
    <span className={clsx('inline-flex items-center gap-1 px-1.5 py-0.5 rounded-sm text-2xs font-semibold whitespace-nowrap', STYLES[status])}>
      {status === 'processing' && <Loader2 size={10} className="animate-spin" aria-hidden />}
      {t(`status.${status}`)}
    </span>
  )
}
