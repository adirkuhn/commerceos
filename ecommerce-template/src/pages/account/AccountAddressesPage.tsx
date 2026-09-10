import { useState } from 'react'
import { Plus, Trash2, Pencil, Star, MapPin } from 'lucide-react'
import { Seo } from '@/components/seo/Seo'
import { PageHeader } from '@/components/layout/PageHeader'
import { Button } from '@/components/ui/Button'
import { Dialog, DialogContent, DialogTitle } from '@/components/ui/Dialog'
import { EmptyState } from '@/components/ui/States'
import { AddressForm } from '@/components/ecommerce/AddressForm'
import { useAuthStore } from '@/stores/authStore'
import { useAddAddress, useUpdateAddress, useRemoveAddress } from '@/features/customers/hooks'
import { toast } from '@/components/ui/toast'
import type { Address } from '@/types'

export default function AccountAddressesPage() {
  const customer = useAuthStore(s => s.session?.customer)
  const addAddress = useAddAddress()
  const updateAddress = useUpdateAddress()
  const removeAddress = useRemoveAddress()
  const [editing, setEditing] = useState<Address | 'new' | null>(null)

  const addresses = customer?.addresses ?? []

  return (
    <>
      <Seo title="Addresses" />
      <PageHeader
        title="Addresses"
        action={<Button size="sm" onClick={() => setEditing('new')}><Plus className="mr-2 h-4 w-4" /> Add address</Button>}
      />

      {addresses.length === 0 ? (
        <EmptyState icon={<MapPin className="h-8 w-8" />} title="No saved addresses" description="Add a billing or shipping address to speed up checkout." />
      ) : (
        <div className="grid gap-4 sm:grid-cols-2">
          {addresses.map(address => (
            <div key={address.id} className="rounded-[var(--radius)] border border-border p-4">
              <div className="mb-2 flex items-center justify-between">
                <p className="text-sm font-medium">{address.label ?? 'Address'}</p>
                <div className="flex gap-1">
                  {(address.isDefaultBilling || address.isDefaultShipping) && <Star className="h-3.5 w-3.5 fill-amber-400 text-amber-400" />}
                  <button aria-label="Edit address" onClick={() => setEditing(address)}><Pencil className="h-3.5 w-3.5 text-muted-foreground hover:text-foreground" /></button>
                  <button aria-label="Delete address" onClick={() => removeAddress.mutate(address.id, { onSuccess: () => toast({ title: 'Address removed' }) })}>
                    <Trash2 className="h-3.5 w-3.5 text-muted-foreground hover:text-destructive" />
                  </button>
                </div>
              </div>
              <p className="text-sm text-muted-foreground">
                {address.firstName} {address.lastName}<br />{address.line1}{address.line2 ? `, ${address.line2}` : ''}<br />
                {address.city}, {address.province} {address.postalCode}<br />{address.country}
              </p>
            </div>
          ))}
        </div>
      )}

      <Dialog open={editing !== null} onOpenChange={(open) => !open && setEditing(null)}>
        <DialogContent>
          <DialogTitle>{editing === 'new' ? 'Add address' : 'Edit address'}</DialogTitle>
          <AddressForm
            defaultValues={editing !== 'new' ? editing ?? undefined : undefined}
            onSubmit={(values) => {
              if (editing === 'new') {
                addAddress.mutate(values, { onSuccess: () => { toast({ title: 'Address added' }); setEditing(null) } })
              } else if (editing) {
                updateAddress.mutate({ ...editing, ...values }, { onSuccess: () => { toast({ title: 'Address updated' }); setEditing(null) } })
              }
            }}
          />
        </DialogContent>
      </Dialog>
    </>
  )
}
