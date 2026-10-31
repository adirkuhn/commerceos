import type { ContactPerson, ContactPersonInput, Customer, CustomerInput, WhatsAppMessage, APIResponse } from './types'
import { CodedApiError } from '../api/codedApiError'
import { appEnv } from '../config/appEnv'

function base(): string {
  return `${appEnv().VITE_API_BASE_URL}/customers`
}

async function request<T>(
  path: string,
  options: RequestInit,
  token: string,
): Promise<T> {
  const res = await fetch(path, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
      ...options.headers,
    },
  })
  const body: APIResponse<T> = await res.json()
  if (!res.ok || body.error) {
    throw CodedApiError.fromEnvelope(body.error, res.status)
  }
  return body.data as T
}

export function listCustomers(token: string): Promise<Customer[]> {
  return request<Customer[]>(`${base()}`, { method: 'GET' }, token)
}

export function getCustomer(token: string, id: string): Promise<Customer> {
  return request<Customer>(`${base()}/${id}`, { method: 'GET' }, token)
}

export function createCustomer(token: string, input: CustomerInput): Promise<Customer> {
  return request<Customer>(`${base()}`, { method: 'POST', body: JSON.stringify(input) }, token)
}

export function updateCustomer(token: string, id: string, input: CustomerInput): Promise<Customer> {
  return request<Customer>(`${base()}/${id}`, { method: 'PATCH', body: JSON.stringify(input) }, token)
}

export function deleteCustomer(token: string, id: string): Promise<null> {
  return request<null>(`${base()}/${id}`, { method: 'DELETE' }, token)
}

export function listCustomerWhatsAppMessages(token: string, id: string): Promise<WhatsAppMessage[]> {
  return request<WhatsAppMessage[]>(`${base()}/${id}/whatsapp-messages`, { method: 'GET' }, token)
}

export function listContactPersons(token: string, customerId: string): Promise<ContactPerson[]> {
  return request<ContactPerson[]>(`${base()}/${customerId}/contact-persons`, { method: 'GET' }, token)
}

export function addContactPerson(token: string, customerId: string, input: ContactPersonInput): Promise<ContactPerson> {
  return request<ContactPerson>(`${base()}/${customerId}/contact-persons`, { method: 'POST', body: JSON.stringify(input) }, token)
}

export function updateContactPerson(token: string, customerId: string, id: string, input: ContactPersonInput): Promise<ContactPerson> {
  return request<ContactPerson>(`${base()}/${customerId}/contact-persons/${id}`, { method: 'PATCH', body: JSON.stringify(input) }, token)
}

export function removeContactPerson(token: string, customerId: string, id: string): Promise<null> {
  return request<null>(`${base()}/${customerId}/contact-persons/${id}`, { method: 'DELETE' }, token)
}
