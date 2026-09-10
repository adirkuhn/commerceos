import type { ReactNode } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { addressSchema, type AddressFormValues } from '@/lib/validation'
import { Input } from '@/components/ui/Input'
import { Label } from '@/components/ui/Label'
import { Button } from '@/components/ui/Button'
import { Select } from '@/components/ui/Select'

const COUNTRIES = [
  { code: 'US', name: 'United States' }, { code: 'CA', name: 'Canada' }, { code: 'GB', name: 'United Kingdom' },
  { code: 'DE', name: 'Germany' }, { code: 'FR', name: 'France' }, { code: 'NL', name: 'Netherlands' }, { code: 'ES', name: 'Spain' },
]

function Field({ id, label, error, children }: { id: string; label: string; error?: string; children: ReactNode }) {
  return (
    <div>
      <Label htmlFor={id}>{label}</Label>
      <div className="mt-1.5">{children}</div>
      {error && <p role="alert" className="mt-1 text-xs text-destructive">{error}</p>}
    </div>
  )
}

export function AddressForm({
  defaultValues, onSubmit, submitLabel = 'Save address', showVatField = false, idPrefix = 'address',
}: {
  defaultValues?: Partial<AddressFormValues>
  onSubmit: (values: AddressFormValues) => void
  submitLabel?: string
  showVatField?: boolean
  idPrefix?: string
}) {
  const { register, handleSubmit, formState: { errors, isSubmitting } } = useForm<AddressFormValues>({
    resolver: zodResolver(addressSchema),
    defaultValues: { country: 'US', ...defaultValues },
  })
  const id = (name: string) => `${idPrefix}-${name}`

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-4" noValidate>
      <div className="grid grid-cols-2 gap-4">
        <Field id={id('first-name')} label="First name" error={errors.firstName?.message}>
          <Input id={id('first-name')} {...register('firstName')} autoComplete="given-name" />
        </Field>
        <Field id={id('last-name')} label="Last name" error={errors.lastName?.message}>
          <Input id={id('last-name')} {...register('lastName')} autoComplete="family-name" />
        </Field>
      </div>
      <Field id={id('company')} label="Company (optional)">
        <Input id={id('company')} {...register('company')} autoComplete="organization" />
      </Field>
      <Field id={id('line1')} label="Street address" error={errors.line1?.message}>
        <Input id={id('line1')} {...register('line1')} autoComplete="address-line1" />
      </Field>
      <Field id={id('line2')} label="Apartment, suite, etc. (optional)">
        <Input id={id('line2')} {...register('line2')} autoComplete="address-line2" />
      </Field>
      <div className="grid grid-cols-3 gap-4">
        <Field id={id('city')} label="City" error={errors.city?.message}>
          <Input id={id('city')} {...register('city')} autoComplete="address-level2" />
        </Field>
        <Field id={id('province')} label="State / province">
          <Input id={id('province')} {...register('province')} autoComplete="address-level1" />
        </Field>
        <Field id={id('postal-code')} label="Postal code" error={errors.postalCode?.message}>
          <Input id={id('postal-code')} {...register('postalCode')} autoComplete="postal-code" />
        </Field>
      </div>
      <div className="grid grid-cols-2 gap-4">
        <Field id={id('country')} label="Country" error={errors.country?.message}>
          <Select id={id('country')} {...register('country')} autoComplete="country">
            {COUNTRIES.map(c => <option key={c.code} value={c.code}>{c.name}</option>)}
          </Select>
        </Field>
        <Field id={id('phone')} label="Phone (optional)">
          <Input id={id('phone')} type="tel" {...register('phone')} autoComplete="tel" />
        </Field>
      </div>
      {showVatField && (
        <Field id={id('vat')} label="VAT / tax ID (optional)">
          <Input id={id('vat')} {...register('vatNumber')} />
        </Field>
      )}
      <Button type="submit" disabled={isSubmitting}>{isSubmitting ? 'Saving…' : submitLabel}</Button>
    </form>
  )
}
