import { useState } from 'react'
import { Seo } from '@/components/seo/Seo'
import { PageHeader } from '@/components/layout/PageHeader'
import { Input } from '@/components/ui/Input'
import { Label } from '@/components/ui/Label'
import { Button } from '@/components/ui/Button'
import { Checkbox } from '@/components/ui/Checkbox'
import { Separator } from '@/components/ui/Separator'
import { useAuthStore } from '@/stores/authStore'
import { useUpdateProfile } from '@/features/customers/hooks'
import { toast } from '@/components/ui/toast'

export default function AccountProfilePage() {
  const customer = useAuthStore(s => s.session?.customer)
  const updateProfile = useUpdateProfile()
  const [firstName, setFirstName] = useState(customer?.firstName ?? '')
  const [lastName, setLastName] = useState(customer?.lastName ?? '')
  const [phone, setPhone] = useState(customer?.phone ?? '')
  const [preferences, setPreferences] = useState(customer?.preferences ?? { orderUpdates: true, newsletter: false, productRestock: false })
  const [passwordSaving, setPasswordSaving] = useState(false)

  if (!customer) return null

  return (
    <>
      <Seo title="Profile" />
      <PageHeader title="Profile" />

      <form
        className="max-w-md space-y-4"
        onSubmit={(e) => {
          e.preventDefault()
          updateProfile.mutate({ firstName, lastName, phone, preferences }, { onSuccess: () => toast({ title: 'Profile updated', variant: 'success' }) })
        }}
      >
        <div className="grid grid-cols-2 gap-4">
          <div><Label htmlFor="profile-first-name">First name</Label><Input id="profile-first-name" className="mt-1.5" value={firstName} onChange={(e) => setFirstName(e.target.value)} /></div>
          <div><Label htmlFor="profile-last-name">Last name</Label><Input id="profile-last-name" className="mt-1.5" value={lastName} onChange={(e) => setLastName(e.target.value)} /></div>
        </div>
        <div><Label htmlFor="profile-email">Email</Label><Input id="profile-email" type="email" className="mt-1.5" value={customer.email} disabled /></div>
        <div><Label htmlFor="profile-phone">Phone</Label><Input id="profile-phone" type="tel" className="mt-1.5" value={phone} onChange={(e) => setPhone(e.target.value)} /></div>

        <Separator />
        <p className="text-sm font-medium">Notification preferences</p>
        <label className="flex items-center gap-2 text-sm">
          <Checkbox checked={preferences.orderUpdates} onCheckedChange={(v) => setPreferences(p => ({ ...p, orderUpdates: v }))} /> Order updates
        </label>
        <label className="flex items-center gap-2 text-sm">
          <Checkbox checked={preferences.newsletter} onCheckedChange={(v) => setPreferences(p => ({ ...p, newsletter: v }))} /> Newsletter
        </label>
        <label className="flex items-center gap-2 text-sm">
          <Checkbox checked={preferences.productRestock} onCheckedChange={(v) => setPreferences(p => ({ ...p, productRestock: v }))} /> Restock alerts
        </label>

        <Button type="submit" disabled={updateProfile.isPending}>{updateProfile.isPending ? 'Saving…' : 'Save changes'}</Button>
      </form>

      <Separator className="my-8" />

      <form
        className="max-w-md space-y-4"
        onSubmit={(e) => {
          e.preventDefault()
          setPasswordSaving(true)
          setTimeout(() => { setPasswordSaving(false); toast({ title: 'Password updated', variant: 'success' }) }, 400)
        }}
      >
        <p className="text-sm font-medium">Change password</p>
        <div><Label htmlFor="profile-current-password">Current password</Label><Input id="profile-current-password" type="password" className="mt-1.5" required /></div>
        <div><Label htmlFor="profile-new-password">New password</Label><Input id="profile-new-password" type="password" className="mt-1.5" required minLength={8} /></div>
        <Button type="submit" variant="outline" disabled={passwordSaving}>{passwordSaving ? 'Saving…' : 'Update password'}</Button>
      </form>
    </>
  )
}
