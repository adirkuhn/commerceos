import { useTranslation } from 'react-i18next'
import { useDateFormat } from '../format/DateFormatProvider'
import type { AssistantRequest } from './types'

// Why a queued request is still waiting after a failed attempt -- nothing
// for a fresh request that simply hasn't been picked up yet.
export function WaitReason({ request, detailed = false }: { request: AssistantRequest; detailed?: boolean }) {
  const { t } = useTranslation('assistant')
  const { formatTime } = useDateFormat()
  if (request.status !== 'queued' || !request.wait_reason) return null
  return (
    <div className="flex flex-col gap-0.5">
      <p className="text-2xs text-warning">{t(`wait.${request.wait_reason}`)}</p>
      {detailed && (
        <>
          <p className="text-2xs text-ink-4">{t('wait.nextTry', { time: formatTime(request.next_attempt_at) })}</p>
          {request.last_error && <p className="text-2xs text-ink-4 font-mono break-words">{request.last_error}</p>}
        </>
      )}
    </div>
  )
}
