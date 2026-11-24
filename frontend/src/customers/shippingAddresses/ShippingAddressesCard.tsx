import { useState } from 'react'
import { Edit3, Plus } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useConfirm } from '../../components/common/useConfirm'
import type { ShippingAddress, ShippingAddressInput } from '../types'
import { oneLineAddress, shippingAddressName } from './addressLine'
import { ShippingAddressForm } from './ShippingAddressForm'

interface ShippingAddressesCardProps {
  addresses: ShippingAddress[]
  error: string | null
  onAdd: (input: ShippingAddressInput) => Promise<void>
  onUpdate: (id: string, input: ShippingAddressInput) => Promise<void>
  onRemove: (id: string) => Promise<void>
  onMakeDefault: (id: string) => Promise<void>
}

// A customer's shipping addresses, the default marked: list, add, edit,
// remove, make default. Each change saves on its own.
export function ShippingAddressesCard({ addresses, error, onAdd, onUpdate, onRemove, onMakeDefault }: ShippingAddressesCardProps) {
  const { t } = useTranslation('customers')
  const { confirm, confirmDialog } = useConfirm()
  // 'new', an address's id, or null when no form is open.
  const [editing, setEditing] = useState<string | null>(null)

  async function remove(a: ShippingAddress) {
    const name = shippingAddressName(a)
    const ok = await confirm({ title: t('shippingAddresses.removeTitle', { name }), message: t('shippingAddresses.removeMessage'), confirmLabel: t('shippingAddresses.remove'), destructive: true })
    if (!ok) return
    await onRemove(a.id)
    setEditing(null)
  }

  return (
    <div className="rounded-xl border border-line p-4 space-y-3 text-xs">
      <h3 className="text-xs font-semibold text-ink">{t('shippingAddresses.title')}</h3>
      {error && <p role="alert" className="text-danger">{error}</p>}
      {addresses.length === 0 && editing !== 'new' && <p className="text-ink-4">{t('shippingAddresses.empty')}</p>}
      <ul className="space-y-2">
        {addresses.map(a => {
          const name = shippingAddressName(a)
          return (
            <li key={a.id} className="rounded-lg border border-line/70 px-3 py-2">
              {editing === a.id ? (
                <ShippingAddressForm initial={a} onCancel={() => setEditing(null)}
                  onSave={async input => { await onUpdate(a.id, input); setEditing(null) }} onRemove={() => remove(a)} />
              ) : (
                <div className="flex items-start justify-between gap-2">
                  <div className="min-w-0">
                    <p className="flex items-center gap-1.5 text-ink">
                      <span className="truncate">{a.label || t('shippingAddresses.untitled')}</span>
                      {a.is_default && <span className="shrink-0 px-1.5 rounded bg-accent/15 text-accent text-2xs font-medium">{t('shippingAddresses.default')}</span>}
                    </p>
                    <p className="text-2xs text-ink-3 truncate">{oneLineAddress(a)}</p>
                  </div>
                  <div className="flex items-center gap-1 shrink-0">
                    {!a.is_default && (
                      <button type="button" onClick={() => void onMakeDefault(a.id)} aria-label={t('shippingAddresses.makeDefaultLabel', { name })}
                        className="px-2 py-0.5 rounded text-2xs text-ink-3 hover:bg-hover">{t('shippingAddresses.makeDefault')}</button>
                    )}
                    <button type="button" onClick={() => setEditing(a.id)} aria-label={t('shippingAddresses.edit', { name })}
                      title={t('shippingAddresses.edit', { name })}
                      className="w-6 h-6 flex items-center justify-center rounded hover:bg-hover"><Edit3 size={12} className="text-ink-3" /></button>
                  </div>
                </div>
              )}
            </li>
          )
        })}
      </ul>
      {editing === 'new' ? (
        <div className="rounded-lg border border-line/70 px-3 py-2">
          <ShippingAddressForm withDefaultChoice onCancel={() => setEditing(null)} onSave={async input => { await onAdd(input); setEditing(null) }} />
        </div>
      ) : (
        <button type="button" onClick={() => setEditing('new')}
          className="w-full flex items-center justify-center gap-1.5 px-3 py-1.5 rounded-lg border border-line text-xs font-medium text-ink-2 hover:bg-hover">
          <Plus size={12} /> {t('shippingAddresses.add')}
        </button>
      )}
      {confirmDialog}
    </div>
  )
}
