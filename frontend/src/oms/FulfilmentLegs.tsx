import clsx from 'clsx'
import { useTranslation } from 'react-i18next'
import { useDateFormat } from '../format/DateFormatProvider'
import { Coupon, CouponHead, type CouponTone } from '../components/common/Coupon'
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

const LEG_TONE: Record<Leg['state'], CouponTone> = { current: 'raised', done: 'plain', upcoming: 'waiting' }

function LegCoupon({ leg, number }: { leg: Leg; number: number }) {
  const { t } = useTranslation('orders')
  const { formatDateTime } = useDateFormat()
  const current = leg.state === 'current'

  return (
    <Coupon
      as="li"
      tone={LEG_TONE[leg.state]}
      stubLabel={t('legs.leg')}
      stubValue={String(number).padStart(2, '0')}
      aria-current={current ? 'step' : undefined}
      className="min-h-[5.5rem]"
    >
      <CouponHead rule={current ? 'accent' : leg.state === 'done' ? 'mark' : 'none'} muted={leg.state === 'upcoming'}>
        {t(`legs.${leg.id}`)}
      </CouponHead>
      {leg.state === 'done' && (
        <span className="self-start -rotate-2 border-2 border-accent/80 rounded-sm px-1.5 py-0.5 caps-label-wide text-2xs font-bold text-accent tabular-nums">
          {t('legs.done')}{leg.at ? ` · ${formatDateTime(leg.at)}` : ''}
        </span>
      )}
      {current && <span className="text-xs font-medium text-accent">{t('legs.upNext')}</span>}
      {leg.state === 'upcoming' && <span className="text-xs text-ink-4">{t('legs.notYet')}</span>}
    </Coupon>
  )
}

// The journey at list-row size: four short segments (done filled, current
// outlined in the accent, upcoming a hairline), struck through when void.
// One accessible name says where the order is.
export function MiniLegs({ order }: { order: LegOrder }) {
  const { t } = useTranslation('orders')
  const { voided, legs } = fulfilmentLegs(order)
  const current = legs.find(l => l.state === 'current')
  const state = voided
    ? t('legs.void')
    : current ? t('legs.legUpNext', { leg: t(`legs.${current.id}`) }) : t('legs.shipped')

  return (
    <span role="img" aria-label={t('legs.summary', { state })} title={state} className="relative inline-flex items-center gap-0.5">
      {legs.map(leg => (
        <span
          key={leg.id}
          className={clsx(
            'h-2 w-3 rounded-[1px]',
            leg.state === 'done' && 'bg-figure',
            leg.state === 'current' && 'border border-accent bg-accent/20',
            leg.state === 'upcoming' && 'bg-line',
          )}
        />
      ))}
      {voided && <span aria-hidden className="absolute inset-x-[-2px] top-1/2 h-0.5 -translate-y-1/2 bg-danger" />}
    </span>
  )
}
