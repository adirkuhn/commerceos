import { useState } from 'react'
import { X } from 'lucide-react'
import type { Address } from '../oms/types'
import type { RecipientAddress } from './types'
import { CARRIERS } from './types'

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

// The "show window to confirm the address, number, complement" step from
// the design -- staff review/edit the MyParcel-shaped structured fields
// (split from the order's own free-text address as a starting point, see
// addressToRecipientDraft) before a label preview is shown. There is no
// MyParcel address-validation endpoint to check against ahead of time; real
// validation happens when MyParcel accepts/rejects the create-shipment call
// itself, surfaced back to the user on LabelPreviewModal's Print step to fix
// and retry.
export function AddressConfirmModal({ initialAddress, defaultCarrier, initialRecipient, initialCarrier, onNext, onClose }: AddressConfirmModalProps) {
  const [recipient, setRecipient] = useState<RecipientAddress>(() => initialRecipient ?? addressToRecipientDraft(initialAddress))
  const [carrier, setCarrier] = useState(initialCarrier ?? defaultCarrier ?? 1)

  function field<K extends keyof RecipientAddress>(key: K) {
    return {
      value: recipient[key] ?? '',
      onChange: (e: React.ChangeEvent<HTMLInputElement>) => setRecipient(prev => ({ ...prev, [key]: e.target.value })),
    }
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
              <input id="addr-number" {...field('number')} className={inputClass} />
            </div>
          </div>
          <div>
            <label htmlFor="addr-number-suffix" className={labelClass}>Complement (toevoeging)</label>
            <input id="addr-number-suffix" {...field('number_suffix')} className={inputClass} />
          </div>
          <div className="grid grid-cols-2 gap-2">
            <div>
              <label htmlFor="addr-postal-code" className={labelClass}>Postal code</label>
              <input id="addr-postal-code" {...field('postal_code')} className={inputClass} />
            </div>
            <div>
              <label htmlFor="addr-city" className={labelClass}>City</label>
              <input id="addr-city" required {...field('city')} className={inputClass} />
            </div>
          </div>
          <div>
            <label htmlFor="addr-cc" className={labelClass}>Country code</label>
            <input id="addr-cc" required maxLength={2} {...field('cc')} className={inputClass} />
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
