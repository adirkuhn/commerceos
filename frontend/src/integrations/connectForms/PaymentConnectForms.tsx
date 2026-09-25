import type { PaymentWebhookState, RevolutConnectFormState, StripeConnectFormState } from '../hooks/useConnectForms'
import { PaymentWebhookSetupStep } from '../PaymentWebhookSetupStep'
import { ConnectFormFooter } from './ConnectFormFooter'

interface PaymentFormProps<F> {
  form: F
  webhook: PaymentWebhookState
  error: string
  starting: boolean
  onDone: () => void
}

// After a payment provider connects, the form becomes the "register this
// URL in your dashboard, paste back its secret" step.
function WebhookStep({ webhook, providerLabel, dashboardHint, onDone }: {
  webhook: PaymentWebhookState & { webhook: { url: string } }
  providerLabel: string
  dashboardHint: string
  onDone: () => void
}) {
  return (
    <PaymentWebhookSetupStep
      providerLabel={providerLabel}
      dashboardHint={dashboardHint}
      webhookUrl={webhook.webhook.url}
      secret={webhook.secret}
      onSecretChange={webhook.setSecret}
      onSubmit={webhook.save}
      saving={webhook.saving}
      error={webhook.error}
      saved={webhook.saved}
      onDone={onDone}
    />
  )
}

function hasWebhook(w: PaymentWebhookState): w is PaymentWebhookState & { webhook: { connectionId: string; url: string } } {
  return w.webhook !== null
}

export function StripeConnectForm({ form, webhook, error, starting, onDone }: PaymentFormProps<StripeConnectFormState>) {
  if (hasWebhook(webhook)) {
    return <WebhookStep webhook={webhook} providerLabel="Stripe" dashboardHint="Stripe Dashboard → Developers → Webhooks" onDone={onDone} />
  }
  return (
    <form onSubmit={form.connect} className="space-y-3">
      <div>
        <label htmlFor="stripe-secret-key" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
          Secret key
        </label>
        <input
          id="stripe-secret-key"
          type="password"
          required
          autoFocus
          value={form.secretKey}
          onChange={e => form.setSecretKey(e.target.value)}
          placeholder="sk_live_..."
          className="w-full px-3 py-2 text-xs font-mono rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus"
        />
        <p className="mt-1.5 text-2xs text-ink-4">
          From your Stripe Dashboard → Developers → API keys.
        </p>
      </div>
      <div>
        <label htmlFor="stripe-publishable-key" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
          Publishable key
        </label>
        <input
          id="stripe-publishable-key"
          type="text"
          value={form.publishableKey}
          onChange={e => form.setPublishableKey(e.target.value)}
          placeholder="pk_live_..."
          className="w-full px-3 py-2 text-xs font-mono rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus"
        />
        <p className="mt-1.5 text-2xs text-ink-4">
          Optional — required only to offer an embedded card form at checkout instead of a hosted payment link.
        </p>
      </div>
      <ConnectFormFooter error={error} starting={starting} />
    </form>
  )
}

export function RevolutConnectForm({ form, webhook, error, starting, onDone }: PaymentFormProps<RevolutConnectFormState>) {
  if (hasWebhook(webhook)) {
    return <WebhookStep webhook={webhook} providerLabel="Revolut" dashboardHint="Revolut Business → Merchant API → Webhooks" onDone={onDone} />
  }
  return (
    <form onSubmit={form.connect} className="space-y-3">
      <div>
        <label htmlFor="revolut-api-key" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
          Merchant API secret key
        </label>
        <input
          id="revolut-api-key"
          type="password"
          required
          autoFocus
          value={form.apiKey}
          onChange={e => form.setApiKey(e.target.value)}
          className="w-full px-3 py-2 text-xs font-mono rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus"
        />
        <p className="mt-1.5 text-2xs text-ink-4">
          From your Revolut Business account → Merchant API settings.
        </p>
      </div>
      <ConnectFormFooter error={error} starting={starting} />
    </form>
  )
}
