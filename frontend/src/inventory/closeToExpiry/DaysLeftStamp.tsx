import clsx from 'clsx'
import { useTranslation } from 'react-i18next'
import { urgencyOf, type Urgency } from './clearance'

// Ink colour per urgency -- the same semantic tokens both designs define.
const TONE: Record<Urgency, string> = {
  critical: 'text-danger border-danger',
  high: 'text-warning border-warning',
  medium: 'text-accent border-accent',
  low: 'text-ink-3 border-ink-4',
}

// The days left before a batch expires, set like a best-before date stamp:
// a double-ruled box with the number large and the unit small, inked in
// the batch's urgency. The one loud element of the close-to-expiry list.
export function DaysLeftStamp({ daysLeft }: { daysLeft: number }) {
  const { t } = useTranslation('inventory')
  const today = daysLeft <= 0
  return (
    <div
      role="img"
      aria-label={today ? t('closeToExpiry.stampLabelToday') : t('closeToExpiry.stampLabel', { count: daysLeft })}
      className={clsx(
        'w-14 shrink-0 -rotate-2 rounded-md border-2 p-px select-none',
        TONE[urgencyOf(daysLeft)],
      )}
    >
      <div className="rounded-[3px] border border-current px-1 py-1 text-center leading-none">
        {today ? (
          <span className="block py-1.5 text-2xs font-bold">{t('closeToExpiry.stampToday')}</span>
        ) : (
          <>
            <span className="block text-lg font-bold tabular-nums">{daysLeft}</span>
            <span className="block mt-0.5 text-2xs font-semibold">{t('closeToExpiry.stampUnit', { count: daysLeft })}</span>
          </>
        )}
      </div>
    </div>
  )
}
