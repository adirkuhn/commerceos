export interface Address {
  id: string
  label?: string
  firstName: string
  lastName: string
  company?: string
  line1: string
  line2?: string
  city: string
  province?: string
  postalCode: string
  country: string
  phone?: string
  vatNumber?: string
  isDefaultBilling?: boolean
  isDefaultShipping?: boolean
}

export interface NotificationPreferences {
  orderUpdates: boolean
  newsletter: boolean
  productRestock: boolean
}

export type CustomerRole = 'owner' | 'buyer' | 'approver'

export interface Company {
  id: string
  name: string
  vatNumber?: string
  priceListId?: string
  purchaseLimitMinorUnits?: number
  requiresApprovalAboveMinorUnits?: number
}

// A company-scoped price list -- a flat discount off every product's list
// price for now (matches the flat-percent style CommerceOS's own admin
// uses for a channel discount). Real per-SKU overrides would extend this
// with a `productOverrides` map without changing callers of
// lib/b2bPricing.ts's applyPriceList.
export interface PriceList {
  id: string
  name: string
  discountPercent: number
}

// One row of a company's user roster (Company Account view) -- distinct
// from Customer itself, since most of a company's users are never the
// customer actually logged into this browser.
export interface CompanyUserSummary {
  id: string
  name: string
  email: string
  role: CustomerRole
}

export interface Customer {
  id: string
  firstName: string
  lastName: string
  email: string
  phone?: string
  addresses: Address[]
  preferences: NotificationPreferences
  company?: Company
  companyRole?: CustomerRole
  createdAt: string
}
