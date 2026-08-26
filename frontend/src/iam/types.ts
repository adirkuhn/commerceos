export interface Group {
  id: string
  tenant_id: string
  name: string
  description: string
  created_at: string
  updated_at: string
}

export interface CreateGroupPayload {
  name: string
  description?: string
}

export interface UpdateGroupPayload {
  name: string
  description?: string
}

export interface AddMemberPayload {
  user_id: string
}

export interface AssignPermissionPayload {
  rule: string
}

export interface MembersResponse {
  members: string[]
}

export interface PermissionsResponse {
  permissions: string[]
}

// UserActivity mirrors oms's SalesOrderActivity -- the same log, sliced by
// acting user instead of by order (no per-user detail page existed before
// this, so it surfaces here as a Users page side panel).
export interface UserActivity {
  id: string
  sales_order_id: string
  user_id: string
  activity_type: string
  summary: string
  metadata: Record<string, unknown>
  created_at: string
}

export interface UserActivitiesResponse {
  activities: UserActivity[]
}

export interface APIError {
  code: string
  message: string
}

export interface APIResponse<T> {
  data: T | null
  error: APIError | null
}
