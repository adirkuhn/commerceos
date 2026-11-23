import type { Connection, CatalogEntry, StartWooConnectResponse, WebhookSecretResult } from '../types'
import { base, request } from './client'

export function listCatalog(token: string): Promise<CatalogEntry[]> {
  return request<CatalogEntry[]>(`${base()}/catalog`, { method: 'GET' }, token)
}

export function listConnections(token: string): Promise<Connection[]> {
  return request<Connection[]>(`${base()}/connections`, { method: 'GET' }, token)
}

export function startWooConnect(token: string, storeUrl: string): Promise<StartWooConnectResponse> {
  return request<StartWooConnectResponse>(
    `${base()}/woocommerce/connect/start`,
    { method: 'POST', body: JSON.stringify({ store_url: storeUrl }) },
    token,
  )
}

export function disconnectConnection(token: string, id: string): Promise<null> {
  return request<null>(`${base()}/connections/${id}`, { method: 'DELETE' }, token)
}

export function testConnection(token: string, id: string): Promise<Connection> {
  return request<Connection>(`${base()}/connections/${id}/test`, { method: 'POST' }, token)
}

export function updateConnectionSettings(
  token: string, connectionId: string, defaultCurrency: string, defaultWarehouseId: string | null = null,
  bolEconomicOperatorId: string | null = null, bolDeliveryCode: string | null = null,
  bolShippingCostMinorUnits?: number,
): Promise<Connection> {
  return request<Connection>(
    `${base()}/connections/${connectionId}/settings`,
    {
      method: 'PATCH',
      body: JSON.stringify({
        default_currency: defaultCurrency, default_warehouse_id: defaultWarehouseId,
        bol_economic_operator_id: bolEconomicOperatorId, bol_delivery_code: bolDeliveryCode,
        bol_shipping_cost_minor_units: bolShippingCostMinorUnits,
      }),
    },
    token,
  )
}

// (Re)generates this connection's webhook-signature secret, or stores one
// the provider itself generated (Stripe/Revolut -- pass `secret`) instead
// of generating one here (MyParcel/WooCommerce/WhatsApp's flow, omit
// `secret`). The response carries the PLAINTEXT value exactly once, it
// can't be retrieved again after this call (only the encrypted form
// persists server-side).
export function setWebhookSecret(token: string, connectionId: string, secret?: string): Promise<WebhookSecretResult> {
  return request<WebhookSecretResult>(
    `${base()}/connections/${connectionId}/webhook-secret`,
    { method: 'POST', body: secret ? JSON.stringify({ secret }) : undefined },
    token,
  )
}

// enabled:true with an empty testPhoneE164 is rejected by the backend
// (422) -- pass a non-empty number whenever enabling.
export function setWhatsAppTestMode(token: string, connectionId: string, enabled: boolean, testPhoneE164: string): Promise<Connection> {
  return request<Connection>(
    `${base()}/connections/${connectionId}/whatsapp-test-mode`,
    { method: 'POST', body: JSON.stringify({ enabled, test_phone_e164: testPhoneE164 }) },
    token,
  )
}

// enabled:true is rejected by the backend (422) unless the tenant already
// has an AI connection configured (Settings' AI card).
export function setWhatsAppAIParsing(token: string, connectionId: string, enabled: boolean): Promise<Connection> {
  return request<Connection>(
    `${base()}/connections/${connectionId}/whatsapp-ai-parsing`,
    { method: 'POST', body: JSON.stringify({ enabled }) },
    token,
  )
}

// Rotates a WhatsApp connection's base URL/instance name/API key in place
// -- unlike connectWhatsapp, this never touches the webhook secret or
// test-mode/AI-parsing settings. apiKey empty/omitted keeps the currently
// stored key unchanged (it's never sent back to prefill the form).
export function updateWhatsAppCredentials(
  token: string,
  connectionId: string,
  input: { baseUrl: string; instanceName: string; apiKey?: string },
): Promise<Connection> {
  return request<Connection>(
    `${base()}/connections/${connectionId}/whatsapp-credentials`,
    { method: 'POST', body: JSON.stringify({ base_url: input.baseUrl, instance_name: input.instanceName, api_key: input.apiKey || undefined }) },
    token,
  )
}

export interface ConnectEmailPayload {
  host: string
  port: string
  username: string
  password: string
  from: string
}

// No OAuth -- paste your own outbound SMTP relay's credentials. Username/
// password optional (some relays need no auth); omitted from the request
// body entirely when blank rather than sent as empty strings.
export function connectEmail(token: string, payload: ConnectEmailPayload): Promise<Connection> {
  return request<Connection>(
    `${base()}/email/connect`,
    {
      method: 'POST',
      body: JSON.stringify({
        host: payload.host,
        port: payload.port,
        username: payload.username || undefined,
        password: payload.password || undefined,
        from: payload.from,
      }),
    },
    token,
  )
}

export interface ConnectBolPayload {
  clientId: string
  clientSecret: string
  economicOperatorId: string
}

// Bol uses client_credentials (client_id/client_secret pasted from Bol's
// Retailer API portal) -- unlike WooCommerce there's no OAuth redirect
// flow, so this is the only connect path.
export function connectBol(token: string, payload: ConnectBolPayload): Promise<Connection> {
  return request<Connection>(
    `${base()}/bol/connect`,
    {
      method: 'POST',
      body: JSON.stringify({
        client_id: payload.clientId,
        client_secret: payload.clientSecret,
        economic_operator_id: payload.economicOperatorId,
      }),
    },
    token,
  )
}

export interface ConnectWhatsAppPayload {
  baseUrl: string
  instanceName: string
  apiKey: string
}

export interface ConnectWhatsAppResult {
  connection: Connection
  webhook_url: string
}

// No OAuth -- paste a self-hosted Evolution API instance's base URL/
// instance name/API key. webhook_url is returned once, plaintext -- paste
// it into Evolution API's own webhook settings (messages.upsert event)
// right away, it's never shown again.
export function connectWhatsapp(token: string, payload: ConnectWhatsAppPayload): Promise<ConnectWhatsAppResult> {
  return request<ConnectWhatsAppResult>(
    `${base()}/whatsapp/connect`,
    {
      method: 'POST',
      body: JSON.stringify({ base_url: payload.baseUrl, instance_name: payload.instanceName, api_key: payload.apiKey }),
    },
    token,
  )
}

export interface ConnectPaymentProviderResult {
  connection: Connection
  webhook_url: string
}

// Stripe/Revolut both take a single secret API key -- no OAuth, same
// paste-a-key pattern as connectBol. webhook_url is where to register a
// webhook endpoint in the provider's OWN dashboard -- reversed from
// connectWhatsapp, the provider generates its own signing secret when you
// do, paste it back via setWebhookSecret's optional secret param.
export function connectStripe(token: string, secretKey: string, publishableKey: string): Promise<ConnectPaymentProviderResult> {
  return request<ConnectPaymentProviderResult>(`${base()}/stripe/connect`, { method: 'POST', body: JSON.stringify({ secret_key: secretKey, publishable_key: publishableKey }) }, token)
}

export function connectRevolut(token: string, apiKey: string): Promise<ConnectPaymentProviderResult> {
  return request<ConnectPaymentProviderResult>(`${base()}/revolut/connect`, { method: 'POST', body: JSON.stringify({ api_key: apiKey }) }, token)
}

// Connected Stripe/Revolut integrations -- powers the WhatsApp sales
// channel's payment-method picker.
export function listPaymentConnections(token: string): Promise<Connection[]> {
  return request<Connection[]>(`${base()}/payment-connections`, { method: 'GET' }, token)
}
