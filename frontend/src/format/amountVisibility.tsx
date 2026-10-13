import { createContext, useCallback, useContext, useMemo, useState, type MouseEvent, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import clsx from 'clsx'
import { formatMoney } from './money'

// Money privacy: inside the admin every amount starts hidden, so opening
// the app with someone looking over your shoulder shows no revenue, totals
// or prices. One click on an amount reveals that amount for this page load
// only; the Topbar's "Show amounts" reveals them all, and that global
// choice is remembered in this browser so a refresh keeps it. Storage is a
// convenience: when it is unavailable amounts simply start hidden.
//
// Without a provider (unit tests, the scanner app, standalone screens)
// amounts show normally.

interface AmountVisibility {
  visible: boolean
  setVisible: (visible: boolean) => void
}

const AmountVisibilityContext = createContext<AmountVisibility>({ visible: true, setVisible: () => {} })

export const AMOUNTS_STORAGE_KEY = 'commerceos.amounts'
const VISIBLE = 'visible'
const HIDDEN = 'hidden'

// The subset of Storage the provider needs, so tests (and a blocked
// localStorage) can supply their own.
export interface AmountStorage {
  getItem(key: string): string | null
  setItem(key: string, value: string): void
}

function browserStorage(): AmountStorage | null {
  try {
    return window.localStorage
  } catch {
    return null
  }
}

function loadVisible(storage: AmountStorage | null): boolean {
  try {
    return storage?.getItem(AMOUNTS_STORAGE_KEY) === VISIBLE
  } catch {
    return false
  }
}

function saveVisible(storage: AmountStorage | null, visible: boolean): void {
  try {
    storage?.setItem(AMOUNTS_STORAGE_KEY, visible ? VISIBLE : HIDDEN)
  } catch {
    // Not remembered this time; the in-memory choice still applies.
  }
}

interface AmountVisibilityProviderProps {
  children: ReactNode
  storage?: AmountStorage | null
}

export function AmountVisibilityProvider({ children, storage = browserStorage() }: AmountVisibilityProviderProps) {
  const [visible, setVisibleState] = useState(() => loadVisible(storage))
  const setVisible = useCallback((next: boolean) => {
    setVisibleState(next)
    saveVisible(storage, next)
  }, [storage])
  const value = useMemo(() => ({ visible, setVisible }), [visible, setVisible])
  return <AmountVisibilityContext.Provider value={value}>{children}</AmountVisibilityContext.Provider>
}

export function useAmountVisibility(): AmountVisibility {
  return useContext(AmountVisibilityContext)
}

const MASK = '••••'

// The hidden form of an amount: its currency symbol and a fixed mask, so
// neither the digits nor the number of digits show.
export function maskMoney(currencyCode: string, fallbackCurrency = 'USD'): string {
  try {
    const parts = new Intl.NumberFormat(undefined, { style: 'currency', currency: currencyCode || fallbackCurrency }).formatToParts(0)
    let numberDone = false
    return parts.map(p => {
      if (p.type === 'currency' || p.type === 'literal') return p.value
      if (numberDone) return ''
      numberDone = true
      return MASK
    }).join('')
  } catch {
    return `${MASK} ${currencyCode}`
  }
}

// For amounts that end up in text rather than a component (chart tooltips,
// aria labels, sentences): masked while amounts are hidden.
export function useMoneyFormat(): (minorUnits: number, currencyCode: string, fallbackCurrency?: string) => string {
  const { visible } = useAmountVisibility()
  return useCallback(
    (minorUnits, currencyCode, fallbackCurrency) => (visible
      ? formatMoney(minorUnits, currencyCode, fallbackCurrency)
      : maskMoney(currencyCode, fallbackCurrency)),
    [visible],
  )
}

interface MoneyProps {
  minor: number
  currency: string
  fallbackCurrency?: string
  className?: string
}

// An amount on screen. Hidden amounts render as a small button that
// reveals just this one.
export function Money({ minor, currency, fallbackCurrency, className }: MoneyProps) {
  const { visible } = useAmountVisibility()
  const [revealed, setRevealed] = useState(false)
  const { t } = useTranslation('ui')
  const text = formatMoney(minor, currency, fallbackCurrency)

  if (visible) return <span className={className}>{text}</span>

  function toggle(e: MouseEvent) {
    // Amounts sit inside clickable rows and cards -- revealing one must
    // not also open the row.
    e.stopPropagation()
    setRevealed(r => !r)
  }

  return (
    <button
      type="button"
      onClick={toggle}
      aria-label={revealed ? undefined : t('amounts.reveal')}
      title={revealed ? t('amounts.hideOne') : t('amounts.reveal')}
      aria-pressed={revealed}
      className={clsx('rounded px-0.5 -mx-0.5 hover:bg-hover tabular-nums', !revealed && 'text-ink-4 tracking-wider', className)}
    >
      {revealed ? text : maskMoney(currency, fallbackCurrency)}
    </button>
  )
}
