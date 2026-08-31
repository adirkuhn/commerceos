import { useState } from 'react'
import { X, AlertTriangle } from 'lucide-react'
import type { Address } from '../oms/types'
import type { RecipientAddress } from './types'
import { CARRIERS } from './types'
import * as shippingApi from './api'

// splitStreetAndNumber is a best-effort starting point ONLY -- e.g.
// "Damrak 1A" -> {street:"Damrak", number:"1", suffix:"A"} -- never trusted
// as-is for a real carrier submission. AddressConfirmModal always requires
// the user to review/edit before confirming; this just saves retyping the
// common case where it guesses right.
export function splitStreetAndNumber(address1: string): { street: string; number: string; suffix: string } {
  const match = address1.trim().match(/^(.*?)\s+(\d+)\s*([a-zA-Z0-9-]*)$/)
  if (!match) return { street: address1.trim(), number: '', suffix: '' }
  return { street: match[1], number: match[2], suffix: match[3] ?? '' }
}

export function addressToRecipientDraft(address: Address | undefined): RecipientAddress {
  const { street, number, suffix } = splitStreetAndNumber(address?.address_1 ?? '')
  return {
    person: [address?.first_name, address?.last_name].filter(Boolean).join(' '),
    company: address?.company ?? '',
    street,
    number,
    number_suffix: suffix,
    postal_code: address?.postcode ?? '',
    city: address?.city ?? '',
    cc: (address?.country ?? '').toUpperCase(),
    email: address?.email ?? '',
    phone: address?.phone ?? '',
  }
}

interface AddressConfirmModalProps {
  // Threaded explicitly (not read from AuthContext) -- matches how this
  // modal's two call sites (Picking.tsx via useAuth, PickingScreen.tsx via
  // its own token prop) already have it in scope, and keeps this component
  // usable outside an AuthProvider (see LineItemThumb's identical
  // convention).
  token: string
  initialAddress?: Address
  defaultCarrier?: number
  // Present when returning here from LabelPreviewModal's "Back to edit" --
  // takes priority over initialAddress so a staff edit already made isn't
  // silently discarded by re-deriving from the order's raw address again.
  initialRecipient?: RecipientAddress
  initialCarrier?: number
  // Synchronous: this step is preview-only now -- no shipment is created
  // here. onNext just hands the confirmed fields up to LabelPreviewModal,
  // which is where "Print label" actually creates it (see that component's
  // doc comment for why creation moved to the print click).
  onNext: (input: { carrier: number; recipient: RecipientAddress }) => void
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
export function AddressConfirmModal({ token, initialAddress, defaultCarrier, initialRecipient, initialCarrier, onNext, onClose }: AddressConfirmModalProps) {
  const [recipient, setRecipient] = useState<RecipientAddress>(() => initialRecipient ?? addressToRecipientDraft(initialAddress))
  const [carrier, setCarrier] = useState(initialCarrier ?? defaultCarrier ?? 1)
  const [addressCheck, setAddressCheck] = useState<
    | { status: 'idle' | 'checking' | 'match' | 'not_found' | 'error' }
    | { status: 'mismatch'; suggested: { street: string; city: string } }
  >({ status: 'idle' })

  function field<K extends keyof RecipientAddress>(key: K) {
    return {
      value: recipient[key] ?? '',
      onChange: (e: React.ChangeEvent<HTMLInputElement>) => setRecipient(prev => ({ ...prev, [key]: e.target.value })),
    }
  }

  async function runAddressCheck() {
    if (!token || recipient.cc !== 'NL' || !recipient.postal_code || !recipient.number) return
    setAddressCheck({ status: 'checking' })
    try {
      const houseNumber = `${recipient.number}${recipient.number_suffix ?? ''}`
      const found = await shippingApi.lookupAddress(token, recipient.postal_code, houseNumber)
      if (!found) {
        setAddressCheck({ status: 'not_found' })
        return
      }
      const streetOk = !recipient.street || normalize(recipient.street) === normalize(found.street)
      const cityOk = !recipient.city || normalize(recipient.city) === normalize(found.city)
      if (streetOk && cityOk) {
        // Blank fields get filled silently; already-matching ones just confirm.
        setRecipient(prev => ({ ...prev, street: prev.street || found.street, city: prev.city || found.city }))
        setAddressCheck({ status: 'match' })
      } else {
        setAddressCheck({ status: 'mismatch', suggested: found })
      }
    } catch {
      setAddressCheck({ status: 'error' })
    }
  }

  function useSuggestedAddress() {
    if (addressCheck.status !== 'mismatch') return
    setRecipient(prev => ({ ...prev, street: addressCheck.suggested.street, city: addressCheck.suggested.city }))
    setAddressCheck({ status: 'match' })
  }

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    onNext({ carrier, recipient })
  }

  const inputClass = 'w-full px-2.5 py-1.5 text-xs rounded-lg dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-100 text-gray-900'
  const labelClass = 'block text-[11px] font-medium dark:text-gray-400 text-gray-500 mb-1'

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 dark:bg-black/60 bg-black/30" onClick={onClose} />
      <div className="relative w-full max-w-md dark:bg-[#1F2937] bg-white rounded-xl shadow-2xl border dark:border-[#2D3748] border-gray-200 p-5 max-h-[90vh] overflow-y-auto">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-sm font-semibold dark:text-gray-100 text-gray-900">Confirm shipping address</h3>
          <button onClick={onClose} aria-label="Close"><X size={14} className="dark:text-gray-500 text-gray-400" /></button>
        </div>
        <form onSubmit={handleSubmit} className="space-y-3">
          <div>
            <label htmlFor="addr-person" className={labelClass}>Name</label>
            <input id="addr-person" required {...field('person')} className={inputClass} />
          </div>
          <div className="grid grid-cols-3 gap-2">
            <div className="col-span-2">
              <label htmlFor="addr-street" className={labelClass}>Street</label>
              <input id="addr-street" required {...field('street')} className={inputClass} />
            </div>
            <div>
              <label htmlFor="addr-number" className={labelClass}>Number</label>
              <input id="addr-number" {...field('number')} onBlur={runAddressCheck} className={inputClass} />
            </div>
          </div>
          <div>
            <label htmlFor="addr-number-suffix" className={labelClass}>Complement (toevoeging)</label>
            <input id="addr-number-suffix" {...field('number_suffix')} onBlur={runAddressCheck} className={inputClass} />
          </div>
          <div className="grid grid-cols-2 gap-2">
            <div>
              <label htmlFor="addr-postal-code" className={labelClass}>Postal code</label>
              <input id="addr-postal-code" {...field('postal_code')} onBlur={runAddressCheck} className={inputClass} />
            </div>
            <div>
              <label htmlFor="addr-city" className={labelClass}>City</label>
              <input id="addr-city" required {...field('city')} className={inputClass} />
            </div>
          </div>

          {addressCheck.status === 'checking' && (
            <p className="text-[11px] dark:text-gray-500 text-gray-400">Checking address…</p>
          )}
          {addressCheck.status === 'not_found' && (
            <p className="text-[11px] text-amber-500">No match found for that postal code/number — enter manually.</p>
          )}
          {addressCheck.status === 'error' && (
            <p className="text-[11px] dark:text-gray-500 text-gray-400">Couldn't verify the address right now.</p>
          )}
          {addressCheck.status === 'mismatch' && (
            <div className="flex items-start gap-2 rounded-lg px-3 py-2 bg-amber-50 dark:bg-amber-500/10">
              <AlertTriangle size={13} className="text-amber-500 mt-0.5 shrink-0" />
              <div className="flex-1 text-[11px] text-amber-600 dark:text-amber-400">
                <p>Possible typo: you entered <strong>{recipient.street}, {recipient.city}</strong>, the registry has <strong>{addressCheck.suggested.street}, {addressCheck.suggested.city}</strong>.</p>
                <div className="flex gap-2 mt-1.5">
                  <button type="button" onClick={useSuggestedAddress} className="px-2 py-1 rounded bg-amber-500 text-white text-[11px] font-medium hover:bg-amber-600 transition-colors">
                    Use suggested
                  </button>
                  <button type="button" onClick={() => setAddressCheck({ status: 'idle' })} className="px-2 py-1 rounded dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 text-[11px] dark:text-gray-300 text-gray-600 hover:dark:bg-[#2D3748] hover:bg-gray-50 transition-colors">
                    Keep mine
                  </button>
                </div>
              </div>
            </div>
          )}

          <div>
            <label htmlFor="addr-cc" className={labelClass}>Country code</label>
            <input id="addr-cc" required maxLength={2} {...field('cc')} onBlur={runAddressCheck} className={inputClass} />
          </div>
          <div>
            <label htmlFor="ship-carrier" className={labelClass}>Carrier</label>
            <select id="ship-carrier" value={carrier} onChange={e => setCarrier(Number(e.target.value))} className={inputClass}>
              {CARRIERS.map(c => <option key={c.id} value={c.id}>{c.label}</option>)}
            </select>
          </div>
          <button
            type="submit"
            className="w-full px-4 py-2 rounded-lg bg-blue-600 text-white text-xs font-medium hover:bg-blue-700 transition-colors"
          >
            Review label
          </button>
        </form>
      </div>
    </div>
  )
}
