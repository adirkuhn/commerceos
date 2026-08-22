export interface APIError {
  error: string
  code?: string
  message: string
}

export interface APIResponse<T> {
  data: T
}

export type MemberRole = 'customer_owner' | 'customer_user'

export interface CompanyMember {
  membership_id: string
  user_id: string
  email: string
  first_name: string
  last_name: string
  role: MemberRole
  status: string
  last_login_at: string | null
}

export interface CompanyInvite {
  id: string
  email: string
  role: MemberRole
  expires_at: string
  created_at: string
}
