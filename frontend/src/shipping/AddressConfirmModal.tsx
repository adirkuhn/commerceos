import { useState, useRef, useEffect } from 'react'
import { X, AlertTriangle } from 'lucide-react'
import { Trans, useTranslation } from 'react-i18next'
import type { Address } from '../oms/types'
import type { RecipientAddress, DeliveryPriceOption } from './types'
import { CARRIERS, PACKAGE_TYPES } from './types'
import * as shippingApi from './api'
import { Dialog } from '../components/common/Dialog'
import { formatMoney } from '../format/money'

// ESTIMATE_DEBOUNCE_MS: same purpose as the address check's blur-triggered
// timing, but this one fires on every keystroke-driven state change (no
// blur handler), so it needs an actual debounce -- not just coalescing --
// to avoid a request per keystroke while staff are still typing the postal
// code/number.
const ESTIMATE_DEBOUNCE_MS = 300

// splitStreetAndNumber is a best-effort starting point ONLY -- e.g.
// "Damrak 1A" -> {street:"Damrak", number:"1", suffix:"A"} -- never trusted
// as-is for a real carrier submission. AddressConfirmModal always requires
// the user to review/edit before confirming; this just saves retyping the
// common case where it guesses right.
//
// address2 is a fallback for orders where the house number never ended up
// in address1 at all -- some NL WooCommerce checkouts/address plugins
// split street and house number into separate address lines (e.g.
// address_1: "Fossa Iberica", address_2: "9") rather than one combined
// string. Only used when address1 has no trailing number of its own AND
// address2 is *just* a number (optionally with a letter suffix) -- if
// address2 holds anything else (an apartment/unit line, say), it's left
// alone rather than guessed at.
export function splitStreetAndNumber(address1: string, address2?: string): { street: string; number: string; suffix: string } {
  const trimmed1 = address1.trim()
  const match = trimmed1.match(/^(.*?)\s+(\d+)\s*([a-zA-Z0-9-]*)$/)
  if (match) return { street: match[1], number: match[2], suffix: match[3] ?? '' }

  const address2Match = (address2 ?? '').trim().match(/^(\d+)\s*([a-zA-Z0-9-]*)$/)
  if (address2Match) return { street: trimmed1, number: address2Match[1], suffix: address2Match[2] ?? '' }

  return { street: trimmed1, number: '', suffix: '' }
}

// fallbackEmail is the order's billing email -- WooCommerce orders
// reliably have a billing email (the checkout email) but the shipping
// address often doesn't carry its own, which used to leave this blank and
// reach MyParcel empty (MyParcel rejects some carriers' shipments without
// one). Only used when the shipping address itself has none.
export function addressToRecipientDraft(address: Address | undefined, fallbackEmail?: string): RecipientAddress {
  const { street, number, suffix } = splitStreetAndNumber(address?.address_1 ?? '', address?.address_2)
  return {
    person: [address?.first_name, address?.last_name].filter(Boolean).join(' '),
    company: address?.company ?? '',
    street,
    number,
    number_suffix: suffix,
    postal_code: address?.postcode ?? '',
    city: address?.city ?? '',
    cc: (address?.country ?? '').toUpperCase(),
    email: address?.email || fallbackEmail || '',
    phone: address?.phone ?? '',
  }
}

interface AddressConfirmModalProps {
  // Threaded explicitly (not read from AuthContext) -- matches how this
  // modal's two call sites (Picking.tsx via useAuth, PickingScreen.tsx via
  // its own token prop) already have it in scope, and keeps this component
  // usable outside an AuthProvider (see ProductThumb's identical
  // convention).
  token: string
  initialAddress?: Address
  // See addressToRecipientDraft's doc comment -- the order's billing email,
  // used only when initialAddress itself has none.
  fallbackEmail?: string
  defaultCarrier?: number
  // Connection's own default box type (ShippingConnectionSettings.default_
  // package_type) -- same fallback role as defaultCarrier. Undefined (no
  // connection/settings loaded yet) falls back to Package (1).
  defaultPackageType?: number
  // Needed to fetch the estimated label cost (MyParcel's GET
  // /delivery_options is scoped to a connection's own API key) -- undefined
  // simply skips the estimate entirely, same as a non-NL/BE address.
  connectionId?: string
  // Present when returning here from LabelPreviewModal's "Back to edit" --
  // takes priority over initialAddress so a staff edit already made isn't
  // silently discarded by re-deriving from the order's raw address again.
  initialRecipient?: RecipientAddress
  initialCarrier?: number
  initialPackageType?: number
  // Synchronous: this step is preview-only now -- no shipment is created
  // here. onNext just hands the confirmed fields up to LabelPreviewModal,
  // which is where "Print label" actually creates it (see that component's
  // doc comment for why creation moved to the print click).
  onNext: (input: { carrier: number; packageType: number; recipient: RecipientAddress }) => void
  onClose: () => void
}

function normalize(s: string): string {
  return s.trim().toLowerCase()
}

// The "show window to confirm the address, number, complement" step from
// the design -- staff review/edit the MyParcel-shaped structured fields
// (split from the order's own free-text address as a starting point, see
// addressToRecipientDraft) before a label preview is shown. MyParcel has
// no address-validation endpoint reachable with a merchant API key; once
// postal code + house number are both filled (NL only), this checks them
// against PDOK's public Dutch address registry instead -- auto-filling
// blank street/city, or flagging a mismatch (typo, wrong number) against
// whatever staff already entered, with a one-click fix. Real MyParcel-side
// validation still only happens when the create-shipment call itself
// accepts/rejects it, surfaced on LabelPreviewModal's Print step.
export function AddressConfirmModal({ token, initialAddress, fallbackEmail, defaultCarrier, defaultPackageType, connectionId, initialRecipient, initialCarrier, initialPackageType, onNext, onClose }: AddressConfirmModalProps) {
  const [recipient, setRecipient] = useState<RecipientAddress>(() => initialRecipient ?? addressToRecipientDraft(initialAddress, fallbackEmail))
  const { t } = useTranslation(['shipping', 'common'])
  const [carrier, setCarrier] = useState(initialCarrier ?? defaultCarrier ?? 1)
  const [packageType, setPackageType] = useState(initialPackageType ?? defaultPackageType ?? 1)

  // Estimated label cost -- MyParcel's own GET /delivery_options is the
  // only price MyParcel's API exposes before a shipment exists, and it only
  // supports NL/BE addresses (see backend's
  // ErrDeliveryEstimateUnsupportedCountry) -- every other country stays
  // 'idle' forever, silently showing nothing rather than an "unsupported"
  // message on what would otherwise be most shipments.
  const [estimate, setEstimate] = useState<{ status: 'idle' | 'loading' | 'ready' | 'error'; options: DeliveryPriceOption[] }>({ status: 'idle', options: [] })

  useEffect(() => {
    const cc = recipient.cc.toUpperCase()
    if (!connectionId || !token || (cc !== 'NL' && cc !== 'BE') || !recipient.postal_code || !recipient.number) {
      setEstimate({ status: 'idle', options: [] })
      return
    }
    let cancelled = false
    setEstimate(prev => ({ ...prev, status: 'loading' }))
    const timer = setTimeout(() => {
      shippingApi.getDeliveryPriceEstimate(token, {
        connection_id: connectionId, cc, postal_code: recipient.postal_code ?? '', number: recipient.number ?? '', carrier,
      })
        .then(options => { if (!cancelled) setEstimate({ status: 'ready', options }) })
        .catch(() => { if (!cancelled) setEstimate({ status: 'error', options: [] }) })
    }, ESTIMATE_DEBOUNCE_MS)
    return () => { cancelled = true; clearTimeout(timer) }
  }, [connectionId, token, recipient.cc, recipient.postal_code, recipient.number, carrier])

  const cheapestEstimate = estimate.options.length > 0
    ? estimate.options.reduce((min, o) => (o.amount_minor_units < min.amount_minor_units ? o : min))
    : null
  type AddressCheckState =
    | { status: 'idle' | 'checking' | 'match' | 'not_found' | 'error' | 'dismissed' }
    | { status: 'mismatch'; suggested: { street: string; city: string } }
  const [addressCheck, setAddressCheck] = useState<AddressCheckState>({ status: 'idle' })
  // Deliberately separate from addressCheck.status: that can flip to
  // 'checking' from a blur-triggered lookup at any moment, including the
  // instant a submit click lands (blur fires first as focus leaves the
  // last-edited field) -- tying the submit button's disabled state to it
  // could disable the button between mousedown and click, silently
  // swallowing that exact click. This only reflects a check
  // handleSubmit itself is waiting on.
  const [submitting, setSubmitting] = useState(false)

  function field<K extends keyof RecipientAddress>(key: K) {
    return {
      value: recipient[key] ?? '',
      onChange: (e: React.ChangeEvent<HTMLInputElement>) => {
        setRecipient(prev => ({ ...prev, [key]: e.target.value }))
        // Any edit to a field the check depends on invalidates whatever it
        // last found -- a fresh check (blur, or the submit-time gate) must
        // re-run, not reuse a stale match/dismissal from before the edit.
        if (['street', 'city', 'postal_code', 'number', 'number_suffix', 'cc'].includes(key)) {
          setAddressCheck({ status: 'idle' })
        }
      },
    }
  }

  // Both the blur handlers below and handleSubmit can trigger a check --
  // in a real browser, blurring the last-edited field and clicking
  // "Review label" happen back-to-back, so without coalescing, submit
  // could race a blur-triggered check already in flight (whichever one's
  // setState lost the race would make handleSubmit read a stale
  // 'idle'/'checking' status and skip gating on the real outcome).
  // Sharing one in-flight promise means both callers always await the
  // exact same result instead of each starting/reading their own.
  const inFlightCheckRef = useRef<Promise<AddressCheckState['status']> | null>(null)

  async function runAddressCheckImpl(): Promise<AddressCheckState['status']> {
    if (!token || recipient.cc !== 'NL' || !recipient.postal_code || !recipient.number) return 'idle'
    setAddressCheck({ status: 'checking' })
    try {
      const houseNumber = `${recipient.number}${recipient.number_suffix ?? ''}`
      const found = await shippingApi.lookupAddress(token, recipient.postal_code, houseNumber)
      if (!found) {
        setAddressCheck({ status: 'not_found' })
        return 'not_found'
      }
      const streetOk = !recipient.street || normalize(recipient.street) === normalize(found.street)
      const cityOk = !recipient.city || normalize(recipient.city) === normalize(found.city)
      if (streetOk && cityOk) {
        // Blank fields get filled silently; already-matching ones just confirm.
        setRecipient(prev => ({ ...prev, street: prev.street || found.street, city: prev.city || found.city }))
        setAddressCheck({ status: 'match' })
        return 'match'
      }
      setAddressCheck({ status: 'mismatch', suggested: found })
      return 'mismatch'
    } catch {
      setAddressCheck({ status: 'error' })
      return 'error'
    }
  }

  // Returns the final status so callers can gate on it synchronously --
  // handleSubmit needs to know the outcome of a check it just triggered
  // (or joined), not just that state was (asynchronously) updated
  // somewhere.
  function runAddressCheck(): Promise<AddressCheckState['status']> {
    if (!inFlightCheckRef.current) {
      inFlightCheckRef.current = runAddressCheckImpl().finally(() => { inFlightCheckRef.current = null })
    }
    return inFlightCheckRef.current
  }

  function useSuggestedAddress() {
    if (addressCheck.status !== 'mismatch') return
    setRecipient(prev => ({ ...prev, street: addressCheck.suggested.street, city: addressCheck.suggested.city }))
    setAddressCheck({ status: 'match' })
  }

  // "Validate the address before creating the label": if the check hasn't
  // settled for the current fields yet (idle -- never ran, e.g. a
  // prefilled order staff never touched; or checking -- a blur-triggered
  // one is already in flight), join/run it and gate on the result. An
  // already-resolved status (match/not_found/error/dismissed) proceeds
  // without re-checking; a live, unresolved mismatch always blocks until
  // staff picks "Use suggested" or "Keep mine".
  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (addressCheck.status === 'mismatch') return
    if (
      (addressCheck.status === 'idle' || addressCheck.status === 'checking')
      && recipient.cc === 'NL' && recipient.postal_code && recipient.number
    ) {
      setSubmitting(true)
      const outcome = await runAddressCheck().finally(() => setSubmitting(false))
      if (outcome === 'mismatch') return
    }
    onNext({ carrier, packageType, recipient })
  }

  const inputClass = 'w-full px-2.5 py-1.5 text-xs rounded-lg bg-surface border border-line text-ink'
  const labelClass = 'block text-2xs font-medium text-ink-3 mb-1'

  return (
    <Dialog onClose={onClose} className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 dark:bg-black/60 bg-black/30" onClick={onClose} />
      <div className="relative w-full max-w-md bg-raised rounded-xl shadow-2xl border border-line p-5 max-h-[90vh] overflow-y-auto">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-sm font-semibold text-ink">{t('address.title')}</h3>
          <button onClick={onClose} aria-label={t('common:actions.close')}><X size={14} className="text-ink-4" /></button>
        </div>
        <form onSubmit={handleSubmit} className="space-y-3">
          <div>
            <label htmlFor="addr-person" className={labelClass}>{t('address.name')}</label>
            <input id="addr-person" required {...field('person')} className={inputClass} />
          </div>
          <div className="grid grid-cols-3 gap-2">
            <div className="col-span-2">
              <label htmlFor="addr-street" className={labelClass}>{t('address.street')}</label>
              <input id="addr-street" required {...field('street')} className={inputClass} />
            </div>
            <div>
              <label htmlFor="addr-number" className={labelClass}>{t('address.number')}</label>
              <input id="addr-number" {...field('number')} onBlur={runAddressCheck} className={inputClass} />
            </div>
          </div>
          <div>
            <label htmlFor="addr-number-suffix" className={labelClass}>{t('address.suffix')}</label>
            <input id="addr-number-suffix" {...field('number_suffix')} onBlur={runAddressCheck} className={inputClass} />
          </div>
          <div className="grid grid-cols-2 gap-2">
            <div>
              <label htmlFor="addr-postal-code" className={labelClass}>{t('address.postalCode')}</label>
              <input id="addr-postal-code" {...field('postal_code')} onBlur={runAddressCheck} className={inputClass} />
            </div>
            <div>
              <label htmlFor="addr-city" className={labelClass}>{t('address.city')}</label>
              <input id="addr-city" required {...field('city')} className={inputClass} />
            </div>
          </div>

          {addressCheck.status === 'checking' && (
            <p className="text-2xs text-ink-4">{t('address.checking')}</p>
          )}
          {addressCheck.status === 'not_found' && (
            <p className="text-2xs text-warning">{t('address.notFound')}</p>
          )}
          {addressCheck.status === 'error' && (
            <p className="text-2xs text-ink-4">{t('address.checkError')}</p>
          )}
          {addressCheck.status === 'mismatch' && (
            <div className="flex items-start gap-2 rounded-lg px-3 py-2 bg-amber-50 dark:bg-amber-500/10">
              <AlertTriangle size={13} className="text-warning mt-0.5 shrink-0" />
              <div className="flex-1 text-2xs text-warning">
                <p><Trans t={t} i18nKey="address.mismatch" values={{ entered: `${recipient.street}, ${recipient.city}`, suggested: `${addressCheck.suggested.street}, ${addressCheck.suggested.city}` }} components={{ entered: <strong />, suggested: <strong /> }} /></p>
                <div className="flex gap-2 mt-1.5">
                  <button type="button" onClick={useSuggestedAddress} className="px-2 py-1 rounded bg-amber-500 text-white text-2xs font-medium hover:bg-amber-600 transition-colors">
                    {t('address.useSuggested')}
                  </button>
                  <button type="button" onClick={() => setAddressCheck({ status: 'dismissed' })} className="px-2 py-1 rounded bg-surface border border-line text-2xs text-ink-2 hover:bg-hover transition-colors">
                    {t('address.keepMine')}
                  </button>
                </div>
              </div>
            </div>
          )}

          <div>
            <label htmlFor="addr-cc" className={labelClass}>{t('address.country')}</label>
            <input id="addr-cc" required maxLength={2} {...field('cc')} onBlur={runAddressCheck} className={inputClass} />
          </div>
          <div>
            <label htmlFor="addr-email" className={labelClass}>{t('address.email')}</label>
            {/* Required, not just recommended -- MyParcel rejects some
                carriers' shipments with a 422 when this is blank (it emails
                the recipient tracking updates for those carriers). Prefilled
                from the order's own address, but that can be empty, so
                staff must be able to see and fill it in here rather than it
                silently reaching MyParcel blank. */}
            <input id="addr-email" type="email" required {...field('email')} className={inputClass} />
          </div>
          <div className="grid grid-cols-2 gap-2">
            <div>
              <label htmlFor="ship-carrier" className={labelClass}>{t('address.carrier')}</label>
              <select id="ship-carrier" value={carrier} onChange={e => setCarrier(Number(e.target.value))} className={inputClass}>
                {CARRIERS.map(c => <option key={c.id} value={c.id}>{c.label}</option>)}
              </select>
            </div>
            <div>
              <label htmlFor="ship-package-type" className={labelClass}>{t('address.packageType')}</label>
              <select id="ship-package-type" value={packageType} onChange={e => setPackageType(Number(e.target.value))} className={inputClass}>
                {PACKAGE_TYPES.map(p => <option key={p.id} value={p.id}>{p.label}</option>)}
              </select>
            </div>
          </div>
          {estimate.status === 'loading' && (
            <p className="text-2xs text-ink-4">{t('address.estimating')}</p>
          )}
          {estimate.status === 'ready' && cheapestEstimate && (
            <p className="text-2xs text-ink-3">
              <Trans t={t} i18nKey="address.estimate" values={{ amount: formatMoney(cheapestEstimate.amount_minor_units, cheapestEstimate.currency, 'EUR'), comment: cheapestEstimate.price_comment }} components={{ strong: <strong /> }} />
            </p>
          )}
          <button
            type="submit"
            disabled={submitting}
            className="w-full px-4 py-2 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover disabled:opacity-50 transition-colors"
          >
            {submitting ? t('address.checking') : t('address.review')}
          </button>
        </form>
      </div>
    </Dialog>
  )
}
