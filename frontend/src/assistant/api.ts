import { CodedApiError } from '../api/codedApiError'
import { appEnv } from '../config/appEnv'
import type { AssistantListQuery, AssistantRequest, AssistantRequestDetail, AssistantRequestList, ConfirmProposalRequest } from './types'

interface Envelope<T> {
  data: T
  error: { code: string; message: string } | null
}

function base(): string {
  return `${appEnv().VITE_API_BASE_URL}/assistant/requests`
}

async function request<T>(url: string, token: string, init: RequestInit = {}): Promise<T> {
  const res = await fetch(url, {
    ...init,
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
  })
  const body: Envelope<T> = await res.json()
  if (!res.ok || body.error) throw CodedApiError.fromEnvelope(body.error, res.status)
  return body.data
}

export function listAssistantRequests(token: string, query: AssistantListQuery = {}): Promise<AssistantRequestList> {
  const q = new URLSearchParams()
  if (query.status) q.set('status', query.status)
  if (query.cursor) q.set('cursor', query.cursor)
  if (query.limit) q.set('limit', String(query.limit))
  const qs = q.toString()
  return request<AssistantRequestList>(qs ? `${base()}?${qs}` : base(), token)
}

export function getAssistantRequest(token: string, id: string): Promise<AssistantRequestDetail> {
  return request<AssistantRequestDetail>(`${base()}/${encodeURIComponent(id)}`, token)
}

export function createAssistantRequest(token: string, text: string): Promise<AssistantRequest> {
  return request<AssistantRequest>(base(), token, { method: 'POST', body: JSON.stringify({ text }) })
}

export function replyToAssistantRequest(token: string, id: string, text: string): Promise<AssistantRequest> {
  return request<AssistantRequest>(`${base()}/${encodeURIComponent(id)}/reply`, token, { method: 'POST', body: JSON.stringify({ text }) })
}

export function cancelAssistantRequest(token: string, id: string): Promise<AssistantRequest> {
  return request<AssistantRequest>(`${base()}/${encodeURIComponent(id)}/cancel`, token, { method: 'POST' })
}

function proposalPath(requestId: string, proposalId: string): string {
  return `${base()}/${encodeURIComponent(requestId)}/proposals/${encodeURIComponent(proposalId)}`
}

export function confirmAssistantProposal(token: string, requestId: string, proposalId: string, body: ConfirmProposalRequest = {}): Promise<AssistantRequestDetail> {
  return request<AssistantRequestDetail>(`${proposalPath(requestId, proposalId)}/confirm`, token, { method: 'POST', body: JSON.stringify(body) })
}

export function rejectAssistantProposal(token: string, requestId: string, proposalId: string): Promise<AssistantRequestDetail> {
  return request<AssistantRequestDetail>(`${proposalPath(requestId, proposalId)}/reject`, token, { method: 'POST' })
}
