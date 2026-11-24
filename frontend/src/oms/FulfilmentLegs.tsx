import clsx from 'clsx'
import { useTranslation } from 'react-i18next'
import { useDateFormat } from '../format/DateFormatProvider'
import { fulfilmentLegs, type Leg, type LegOrder } from './fulfilmentLegs'

// The order's journey as a row of ticket coupons: each leg has a numbered
// stub torn off by a perforation. Done legs carry a stamp, the current leg
// is raised, upcoming legs wait in outline. A void order is stamped VOID
// across the whole strip.
export function FulfilmentLegs({ order }: { order: LegOrder }) {
  const { t } = useTranslation('orders')
  const { voided, legs } = fulfilmentLegs(order)

  return (
    <div className="relative">
      <ol aria-label={t('legs.label')} className={clsx('grid grid-cols-2 lg:grid-cols-4 gap-3', voided && 'opacity-60')}>
        {legs.map((leg, i) => <LegCoupon key={leg.id} leg={leg} number={i + 1} />)}
      </ol>
      {voided && (
        <div className="pointer-events-none absolute inset-0 flex items-center justify-center">
          <span
            title={t('legs.voidHint')}
            className="-rotate-6 border-[3px] border-danger rounded-md px-5 py-1 text-3xl font-black caps-label-widest text-danger bg-raised/80"
          >
            {t('legs.void')}
          </span>
        </div>
      )}
    </div>
  )
}

function LegCoupon({ leg, number }: { leg: Leg; number: number }) {
  const { t } = useTranslation('orders')
  const { formatDateTime } = useDateFormat()
  const current = leg.state === 'current'
  const done = leg.state === 'done'

  return (
    <li
      aria-current={current ? 'step' : undefined}
      className={clsx(
        'relative flex min-h-[5.5rem] rounded-lg transition-[transform,box-shadow] duration-200',
        current && 'border-2 border-accent bg-raised shadow-popover -translate-y-0.5',
        done && 'border border-line bg-raised shadow-panel',
        leg.state === 'upcoming' && 'border border-dashed border-line bg-transparent',
      )}
    >
      {/* Stub: the leg number, torn off along the perforation. */}
      <div
        className={clsx(
          'w-14 shrink-0 flex flex-col items-center justify-center gap-0.5 border-r border-dashed',
          current ? 'bg-accent-solid text-white border-accent' : 'border-line',
        )}
      >
        <span className={clsx('caps-label-widest text-2xs font-semibold', current ? 'text-white/80' : 'text-ink-4')}>{t('legs.leg')}</span>
        <span className={clsx('text-xl font-bold tabular-nums leading-none', current ? 'text-white' : done ? 'text-figure' : 'text-ink-4')}>
          {String(number).padStart(2, '0')}
        </span>
      </div>
      <Notch className="-top-[6px]" current={current} />
      <Notch className="-bottom-[6px]" current={current} />

      <div className="flex-1 min-w-0 px-3 py-2.5 flex flex-col justify-between gap-2">
        <h3 className={clsx('flex items-center gap-2 caps-label-widest text-2xs font-bold', leg.state === 'upcoming' ? 'text-ink-4' : 'text-ink')}>
          <span aria-hidden className={clsx('h-0.5 w-3', current ? 'bg-accent' : done ? 'bg-danger' : 'bg-line')} />
          {t(`legs.${leg.id}`)}
        </h3>
        {done && (
          <span className="self-start -rotate-2 border-2 border-accent/80 rounded-sm px-1.5 py-0.5 caps-label-wide text-2xs font-bold text-accent tabular-nums">
            {t('legs.done')}{leg.at ? ` · ${formatDateTime(leg.at)}` : ''}
          </span>
        )}
        {current && <span className="text-xs font-medium text-accent">{t('legs.upNext')}</span>}
        {leg.state === 'upcoming' && <span className="text-xs text-ink-4">{t('legs.notYet')}</span>}
      </div>
    </li>
  )
}

// A punched notch where the perforation meets the coupon edge.
function Notch({ className, current }: { className: string; current: boolean }) {
  return (
    <span
      aria-hidden
      className={clsx(
        'absolute left-[3.5rem] -translate-x-1/2 w-3 h-3 rounded-full bg-canvas border',
        current ? 'border-accent' : 'border-line',
        className,
      )}
    />
  )
}
