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
