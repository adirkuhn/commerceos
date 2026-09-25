interface PaymentWebhookSetupStepProps {
  providerLabel: string
  dashboardHint: string
  webhookUrl: string
  secret: string
  onSecretChange: (value: string) => void
  onSubmit: (e: React.FormEvent) => void
  saving: boolean
  error: string
  saved: boolean
  onDone: () => void
}

// Reversed from WhatsApp's connect step (there we generate a secret and
// the merchant pastes our URL+secret into Evolution API) -- Stripe/Revolut
// generate their OWN signing secret when the merchant registers this URL
// as a webhook endpoint in their own dashboard, so this step shows the URL
// to register there, then collects THEIR secret to paste back into us.
export function PaymentWebhookSetupStep({
  providerLabel, dashboardHint, webhookUrl, secret, onSecretChange, onSubmit, saving, error, saved, onDone,
}: PaymentWebhookSetupStepProps) {
  return (
    <div className="space-y-3">
      <p className="text-xs text-ink-2">
        Connected! Register this URL as a webhook endpoint in {dashboardHint}, then paste the signing secret it gives you below.
      </p>
      <p className="text-2xs font-mono break-all bg-fill rounded-lg px-2 py-1.5 text-ink-2">
        {webhookUrl}
      </p>
      {saved ? (
        <>
          <p className="text-xs text-success bg-green-50 dark:bg-green-500/10 rounded-lg px-2 py-1.5">Webhook secret saved.</p>
          <button
            onClick={onDone}
            className="w-full py-2 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover transition-colors"
          >
            Done
          </button>
        </>
      ) : (
        <form onSubmit={onSubmit} className="space-y-3">
          <div>
            <label htmlFor="payment-webhook-secret" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
              {providerLabel} webhook signing secret
            </label>
            <input
              id="payment-webhook-secret"
              type="password"
              required
              autoFocus
              value={secret}
              onChange={e => onSecretChange(e.target.value)}
              className="w-full px-3 py-2 text-xs font-mono rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus"
            />
          </div>
          {error && (
            <p className="text-xs text-danger bg-red-50 dark:bg-red-500/10 rounded-lg px-2 py-1.5">{error}</p>
          )}
          <div className="grid grid-cols-2 gap-2">
            <button
              type="button"
              onClick={onDone}
              className="py-2 rounded-lg bg-fill text-xs font-medium text-ink-2 transition-colors"
            >
              Skip for now
            </button>
            <button
              type="submit"
              disabled={saving}
              className="py-2 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover disabled:opacity-50 transition-colors"
            >
              {saving ? 'Saving…' : 'Save'}
            </button>
          </div>
        </form>
      )}
    </div>
  )
}
