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

export interface APIError {
  code: string
  message: string
}

export interface APIResponse<T> {
  data: T | null
  error: APIError | null
}
