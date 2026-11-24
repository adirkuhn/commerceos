import type { ReactNode } from 'react'
import clsx from 'clsx'

// A ticket coupon: a stub (a small caps label over a big figure) torn off
// along a perforation, then the coupon body. The tones carry state:
//   raised   -- the current one: accent outline, filled stub, lifted
//   plain    -- a done or ordinary coupon on the page
//   waiting  -- not reached yet: dashed outline, muted
export type CouponTone = 'raised' | 'plain' | 'waiting'

interface CouponProps {
  stubLabel: string
  stubValue: ReactNode
  tone?: CouponTone
  // Wider stubs for figures longer than two digits.
  stubWidth?: 'narrow' | 'wide'
  // Rendered as this element (li inside a list, div otherwise).
  as?: 'li' | 'div'
  className?: string
  'aria-current'?: 'step' | undefined
  children: ReactNode
}

const STUB_WIDTH = { narrow: 'w-14', wide: 'w-24' }
const NOTCH_LEFT = { narrow: 'left-[3.5rem]', wide: 'left-[6rem]' }

export function Coupon({
  stubLabel, stubValue, tone = 'plain', stubWidth = 'narrow', as: Tag = 'div', className, children, ...rest
}: CouponProps) {
  const raised = tone === 'raised'
  return (
    <Tag
      aria-current={rest['aria-current']}
      className={clsx(
        'relative flex rounded-lg transition-[transform,box-shadow] duration-200',
        raised && 'border-2 border-accent bg-raised shadow-popover -translate-y-0.5',
        tone === 'plain' && 'border border-line bg-raised shadow-panel',
        tone === 'waiting' && 'border border-dashed border-line',
        className,
      )}
    >
      <div
        className={clsx(
          STUB_WIDTH[stubWidth],
          'shrink-0 flex flex-col items-center justify-center gap-0.5 border-r border-dashed px-1',
          raised ? 'bg-accent-solid text-white border-accent' : 'border-line',
        )}
      >
        <span className={clsx('caps-label-widest text-2xs font-semibold', raised ? 'text-white/80' : 'text-ink-4')}>{stubLabel}</span>
        <span className={clsx('font-bold tabular-nums leading-none', stubWidth === 'wide' ? 'text-3xl' : 'text-xl', raised ? 'text-white' : tone === 'waiting' ? 'text-ink-4' : 'text-figure')}>
          {stubValue}
        </span>
      </div>
      <Notch edge="top" left={NOTCH_LEFT[stubWidth]} raised={raised} />
      <Notch edge="bottom" left={NOTCH_LEFT[stubWidth]} raised={raised} />
      <div className="flex-1 min-w-0 px-3 py-2.5 flex flex-col justify-between gap-2">{children}</div>
    </Tag>
  )
}

// A coupon's tab head: tracked caps after a short rule.
export function CouponHead({ children, rule, muted }: { children: ReactNode; rule: 'accent' | 'mark' | 'none'; muted?: boolean }) {
  return (
    <h3 className={clsx('flex items-center gap-2 caps-label-widest text-2xs font-bold', muted ? 'text-ink-4' : 'text-ink')}>
      <span aria-hidden className={clsx('h-0.5 w-3 shrink-0', rule === 'accent' ? 'bg-accent' : rule === 'mark' ? 'bg-danger' : 'bg-line')} />
      {children}
    </h3>
  )
}

// A punched notch where the perforation meets the coupon edge.
function Notch({ edge, left, raised }: { edge: 'top' | 'bottom'; left: string; raised: boolean }) {
  return (
    <span
      aria-hidden
      className={clsx(
        'absolute -translate-x-1/2 w-3 h-3 rounded-full bg-canvas border',
        edge === 'top' ? '-top-[6px]' : '-bottom-[6px]',
        left,
        raised ? 'border-accent' : 'border-line',
      )}
    />
  )
}
