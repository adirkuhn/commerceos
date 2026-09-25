import clsx from 'clsx'
import type { WooConnectFormState } from '../hooks/useConnectForms'
import { ConnectFormFooter } from './ConnectFormFooter'

interface Props {
  form: WooConnectFormState
  error: string
  starting: boolean
}

const MODES = [
  { mode: 'auto', label: 'Connect automatically' },
  { mode: 'manual', label: 'Manual setup' },
] as const

function StoreUrlInput({ form }: { form: WooConnectFormState }) {
  return (
    <input
      type="url"
      required
      autoFocus
      value={form.storeUrl}
      onChange={e => form.setStoreUrl(e.target.value)}
      placeholder="https://mystore.com"
      className="w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus"
    />
  )
}

// WooCommerce: OAuth-style redirect ("automatically") or pasted REST API keys.
export function WooConnectForm({ form, error, starting }: Props) {
  return (
    <>
      <div className="flex gap-1 mb-4 p-0.5 rounded-lg bg-fill">
        {MODES.map(({ mode, label }) => (
          <button aria-pressed={form.mode === mode}
            key={mode}
            type="button"
            onClick={() => form.setMode(mode)}
            className={clsx(
              'flex-1 py-1.5 rounded-md text-xs font-medium transition-colors',
              form.mode === mode
                ? 'bg-accent-solid text-white'
                : 'text-ink-3',
            )}
          >
            {label}
          </button>
        ))}
      </div>

      {form.mode === 'auto' ? (
        <form onSubmit={form.startConnect} className="space-y-3">
          <div>
            <label className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
              Store URL
            </label>
            <StoreUrlInput form={form} />
            <p className="mt-1.5 text-2xs text-ink-4">
              Redirects to your store to approve access. Requires your store to reach this server.
            </p>
          </div>
          <ConnectFormFooter error={error} starting={starting} />
        </form>
      ) : (
        <form onSubmit={form.manualConnect} className="space-y-3">
          <div>
            <label className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
              Store URL
            </label>
            <StoreUrlInput form={form} />
          </div>
          <div>
            <label className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
              Consumer key
            </label>
            <input
              type="text"
              required
              value={form.consumerKey}
              onChange={e => form.setConsumerKey(e.target.value)}
              placeholder="ck_..."
              className="w-full px-3 py-2 text-xs font-mono rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus"
            />
          </div>
          <div>
            <label className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
              Consumer secret
            </label>
            <input
              type="password"
              required
              value={form.consumerSecret}
              onChange={e => form.setConsumerSecret(e.target.value)}
              placeholder="cs_..."
              className="w-full px-3 py-2 text-xs font-mono rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus"
            />
            <p className="mt-1.5 text-2xs text-ink-4">
              Generate under WooCommerce → Settings → Advanced → REST API.
            </p>
          </div>
          <ConnectFormFooter error={error} starting={starting} />
        </form>
      )}
    </>
  )
}
