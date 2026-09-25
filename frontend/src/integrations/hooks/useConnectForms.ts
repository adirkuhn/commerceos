import { useState } from 'react'
import * as api from '../api'

type SubmitHandler = (e: React.FormEvent) => Promise<void>

export interface WooConnectFormState {
  mode: 'auto' | 'manual'
  setMode: (mode: 'auto' | 'manual') => void
  storeUrl: string
  setStoreUrl: (v: string) => void
  consumerKey: string
  setConsumerKey: (v: string) => void
  consumerSecret: string
  setConsumerSecret: (v: string) => void
  startConnect: SubmitHandler
  manualConnect: SubmitHandler
}

export interface WhatsAppConnectFormState {
  baseUrl: string
  setBaseUrl: (v: string) => void
  instanceName: string
  setInstanceName: (v: string) => void
  apiKey: string
  setApiKey: (v: string) => void
  // Set once connected -- shown once, for pasting into Evolution API.
  webhookUrl: string
  connect: SubmitHandler
}

export interface StripeConnectFormState {
  secretKey: string
  setSecretKey: (v: string) => void
  publishableKey: string
  setPublishableKey: (v: string) => void
  connect: SubmitHandler
}

export interface RevolutConnectFormState {
  apiKey: string
  setApiKey: (v: string) => void
  connect: SubmitHandler
}

export interface EmailConnectFormState {
  host: string
  setHost: (v: string) => void
  port: string
  setPort: (v: string) => void
  username: string
  setUsername: (v: string) => void
  password: string
  setPassword: (v: string) => void
  from: string
  setFrom: (v: string) => void
  connect: SubmitHandler
}

export interface PaymentWebhookState {
  webhook: { connectionId: string; url: string } | null
  secret: string
  setSecret: (v: string) => void
  save: SubmitHandler
  saving: boolean
  error: string
  saved: boolean
}

export interface ConnectForms {
  error: string
  starting: boolean
  woo: WooConnectFormState
  whatsapp: WhatsAppConnectFormState
  stripe: StripeConnectFormState
  revolut: RevolutConnectFormState
  email: EmailConnectFormState
  paymentWebhook: PaymentWebhookState
}

const errorMessage = (err: unknown, fallback: string) => (err instanceof Error ? err.message : fallback)

// Field values + submit handlers for every form in the "Browse
// integrations" catalog. Owned by the catalog dialog, so everything resets
// when it closes (close buttons or browser Back), while values survive
// stepping between the catalog and a form. `onConnected` reloads the
// connection list; `onClose` closes the catalog.
export function useConnectForms(token: string, onConnected: () => void, onClose: () => void): ConnectForms {
  const [connectError, setConnectError] = useState('')
  const [starting, setStarting] = useState(false)

  const [connectMode, setConnectMode] = useState<'auto' | 'manual'>('auto')
  const [storeUrl, setStoreUrl] = useState('')
  const [consumerKey, setConsumerKey] = useState('')
  const [consumerSecret, setConsumerSecret] = useState('')

  const [waBaseUrl, setWaBaseUrl] = useState('')
  const [waInstanceName, setWaInstanceName] = useState('')
  const [waApiKey, setWaApiKey] = useState('')
  const [waWebhookUrl, setWaWebhookUrl] = useState('')

  const [stripeSecretKey, setStripeSecretKey] = useState('')
  const [stripePublishableKey, setStripePublishableKey] = useState('')
  const [revolutApiKey, setRevolutApiKey] = useState('')

  const [emailHost, setEmailHost] = useState('')
  const [emailPort, setEmailPort] = useState('587')
  const [emailUsername, setEmailUsername] = useState('')
  const [emailPassword, setEmailPassword] = useState('')
  const [emailFrom, setEmailFrom] = useState('')

  // Set once connectStripe/connectRevolut succeeds -- switches the form to
  // the "register this URL, then paste back the secret it gives you" step.
  // Reversed from WhatsApp's flow (there we generate the secret) since
  // Stripe/Revolut generate their own when a webhook endpoint is registered
  // in their dashboard.
  const [paymentWebhook, setPaymentWebhook] = useState<{ connectionId: string; url: string } | null>(null)
  const [paymentWebhookSecret, setPaymentWebhookSecret] = useState('')
  const [savingPaymentWebhookSecret, setSavingPaymentWebhookSecret] = useState(false)
  const [paymentWebhookSecretError, setPaymentWebhookSecretError] = useState('')
  const [paymentWebhookSecretSaved, setPaymentWebhookSecretSaved] = useState(false)

  // Shared submit plumbing: clear the error, flag in-flight, run `action`.
  // `stayBusyOnSuccess` keeps the button disabled after success (the page
  // is navigating away).
  const submit = (action: () => Promise<void>, fallback: string, stayBusyOnSuccess = false): SubmitHandler =>
    async (e: React.FormEvent) => {
      e.preventDefault()
      if (!token) return
      setConnectError('')
      setStarting(true)
      try {
        await action()
        if (!stayBusyOnSuccess) setStarting(false)
      } catch (err) {
        setConnectError(errorMessage(err, fallback))
        setStarting(false)
      }
    }

  const connectPayment = (connectFn: () => Promise<{ connection: { id: string }; webhook_url: string }>) =>
    submit(async () => {
      const { connection, webhook_url } = await connectFn()
      setPaymentWebhook({ connectionId: connection.id, url: webhook_url })
      onConnected()
    }, 'Failed to connect')

  const savePaymentWebhookSecret = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!token || !paymentWebhook) return
    setPaymentWebhookSecretError('')
    setSavingPaymentWebhookSecret(true)
    try {
      await api.setWebhookSecret(token, paymentWebhook.connectionId, paymentWebhookSecret)
      setPaymentWebhookSecretSaved(true)
    } catch (err) {
      setPaymentWebhookSecretError(errorMessage(err, 'Failed to save webhook secret'))
    } finally {
      setSavingPaymentWebhookSecret(false)
    }
  }

  return {
    error: connectError,
    starting,
    woo: {
      mode: connectMode, setMode: setConnectMode,
      storeUrl, setStoreUrl, consumerKey, setConsumerKey, consumerSecret, setConsumerSecret,
      startConnect: submit(async () => {
        const { authorize_url } = await api.startWooConnect(token, storeUrl)
        window.location.href = authorize_url
      }, 'Failed to start connection', true),
      manualConnect: submit(async () => {
        await api.manualConnectWooCommerce(token, { storeUrl, consumerKey, consumerSecret })
        onConnected()
        onClose()
      }, 'Failed to connect', true),
    },
    whatsapp: {
      baseUrl: waBaseUrl, setBaseUrl: setWaBaseUrl,
      instanceName: waInstanceName, setInstanceName: setWaInstanceName,
      apiKey: waApiKey, setApiKey: setWaApiKey,
      webhookUrl: waWebhookUrl,
      connect: submit(async () => {
        const { webhook_url } = await api.connectWhatsapp(token, { baseUrl: waBaseUrl, instanceName: waInstanceName, apiKey: waApiKey })
        setWaWebhookUrl(webhook_url)
        onConnected()
      }, 'Failed to connect'),
    },
    stripe: {
      secretKey: stripeSecretKey, setSecretKey: setStripeSecretKey,
      publishableKey: stripePublishableKey, setPublishableKey: setStripePublishableKey,
      connect: connectPayment(() => api.connectStripe(token, stripeSecretKey, stripePublishableKey)),
    },
    revolut: {
      apiKey: revolutApiKey, setApiKey: setRevolutApiKey,
      connect: connectPayment(() => api.connectRevolut(token, revolutApiKey)),
    },
    email: {
      host: emailHost, setHost: setEmailHost,
      port: emailPort, setPort: setEmailPort,
      username: emailUsername, setUsername: setEmailUsername,
      password: emailPassword, setPassword: setEmailPassword,
      from: emailFrom, setFrom: setEmailFrom,
      connect: submit(async () => {
        await api.connectEmail(token, { host: emailHost, port: emailPort, username: emailUsername, password: emailPassword, from: emailFrom })
        onClose()
        onConnected()
      }, 'Failed to connect'),
    },
    paymentWebhook: {
      webhook: paymentWebhook,
      secret: paymentWebhookSecret, setSecret: setPaymentWebhookSecret,
      save: savePaymentWebhookSecret,
      saving: savingPaymentWebhookSecret,
      error: paymentWebhookSecretError,
      saved: paymentWebhookSecretSaved,
    },
  }
}
