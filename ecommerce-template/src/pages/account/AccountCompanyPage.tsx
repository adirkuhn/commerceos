import { Link } from 'react-router-dom'
import { Building2, ShieldCheck, Users, Wallet, ArrowRight } from 'lucide-react'
import { Seo } from '@/components/seo/Seo'
import { PageHeader } from '@/components/layout/PageHeader'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { EmptyState } from '@/components/ui/States'
import { useAuthStore } from '@/stores/authStore'
import { PRICE_LISTS, DEMO_COMPANY_USERS } from '@/config/demoData'
import { formatMoney } from '@/lib/utils'
import type { CustomerRole } from '@/types'

const ROLE_LABELS: Record<CustomerRole, string> = { owner: 'Owner', buyer: 'Buyer', approver: 'Approver' }
const ROLE_BADGE_VARIANT: Record<CustomerRole, 'default' | 'secondary' | 'outline'> = { owner: 'default', approver: 'secondary', buyer: 'outline' }

// Company account view -- its own page/route (not folded into
// AccountProfilePage) since it's company-wide state, not this one
// customer's own settings. Reads the shared roster from
// config/demoData.ts's DEMO_COMPANY_USERS; a real backend would list
// users scoped to Company.id instead of a flat demo array.
export default function AccountCompanyPage() {
  const company = useAuthStore(s => s.session?.customer.company)
  const companyRole = useAuthStore(s => s.session?.customer.companyRole)

  if (!company) {
    return (
      <>
        <Seo title="Company account" />
        <PageHeader title="Company account" />
        <EmptyState
          icon={<Building2 className="h-8 w-8" />}
          title="No company account"
          description="This customer isn't associated with a business account. Company accounts unlock price lists, purchase limits, and multi-user roles."
        />
      </>
    )
  }

  const priceList = company.priceListId ? PRICE_LISTS.find(pl => pl.id === company.priceListId) : undefined

  return (
    <>
      <Seo title="Company account" />
      <PageHeader
        title={company.name}
        description={company.vatNumber ? `VAT ${company.vatNumber}` : undefined}
        action={companyRole && <Badge variant={ROLE_BADGE_VARIANT[companyRole]}>Your role: {ROLE_LABELS[companyRole]}</Badge>}
      />

      <div className="grid gap-4 sm:grid-cols-3">
        <div className="rounded-[var(--radius)] border border-border p-4">
          <div className="mb-1 flex items-center gap-2 text-sm font-medium"><Wallet className="h-4 w-4 text-primary" /> Price list</div>
          <p className="text-lg font-semibold">{priceList ? priceList.name : 'Standard pricing'}</p>
          {priceList && <p className="text-xs text-muted-foreground">{priceList.discountPercent}% off every list price</p>}
        </div>
        <div className="rounded-[var(--radius)] border border-border p-4">
          <div className="mb-1 flex items-center gap-2 text-sm font-medium"><ShieldCheck className="h-4 w-4 text-primary" /> Purchase limit</div>
          <p className="text-lg font-semibold">{company.purchaseLimitMinorUnits ? formatMoney(company.purchaseLimitMinorUnits, 'USD') : 'No limit set'}</p>
          <p className="text-xs text-muted-foreground">Per order, across the company</p>
        </div>
        <div className="rounded-[var(--radius)] border border-border p-4">
          <div className="mb-1 flex items-center gap-2 text-sm font-medium"><Users className="h-4 w-4 text-primary" /> Approval threshold</div>
          <p className="text-lg font-semibold">{company.requiresApprovalAboveMinorUnits ? formatMoney(company.requiresApprovalAboveMinorUnits, 'USD') : 'Not required'}</p>
          <p className="text-xs text-muted-foreground">Orders above this need approver sign-off</p>
        </div>
      </div>

      <div className="mt-8 flex items-center justify-between">
        <h2 className="text-sm font-semibold">Company users</h2>
        <Button asChild variant="outline" size="sm">
          <Link to="/account/quick-order">Quick order <ArrowRight className="ml-1.5 h-3.5 w-3.5" /></Link>
        </Button>
      </div>
      <div className="mt-3 overflow-x-auto rounded-[var(--radius)] border border-border">
        <table className="w-full text-left text-sm">
          <thead className="border-b border-border bg-secondary/50 text-xs uppercase tracking-wide text-muted-foreground">
            <tr>
              <th className="px-4 py-3 font-medium">Name</th>
              <th className="px-4 py-3 font-medium">Email</th>
              <th className="px-4 py-3 font-medium">Role</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-border">
            {DEMO_COMPANY_USERS.map(user => (
              <tr key={user.id}>
                <td className="px-4 py-3 font-medium">{user.name}</td>
                <td className="px-4 py-3 text-muted-foreground">{user.email}</td>
                <td className="px-4 py-3"><Badge variant={ROLE_BADGE_VARIANT[user.role]}>{ROLE_LABELS[user.role]}</Badge></td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </>
  )
}
