import { useEffect, useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { useAuth } from '../../auth/AuthContext'

export function LoginFormBlock() {
  const { requestLogin, completeLogin } = useAuth()
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const token = searchParams.get('token')

  const [verifying, setVerifying] = useState(!!token)
  const [verifyError, setVerifyError] = useState<string | null>(null)

  const [email, setEmail] = useState('')
  const [sending, setSending] = useState(false)
  const [sent, setSent] = useState(false)
  const [sendError, setSendError] = useState<string | null>(null)

  useEffect(() => {
    if (!token) return
    setVerifying(true)
    completeLogin(token)
      .then(() => navigate('/account'))
      .catch(e => setVerifyError(e instanceof Error ? e.message : 'Unknown error'))
      .finally(() => setVerifying(false))
    // Only ever run once per token -- completeLogin/navigate identity
    // changing shouldn't re-trigger a second redemption attempt of a
    // single-use link.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [token])

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setSendError(null)
    setSending(true)
    try {
      await requestLogin(email, `${window.location.origin}/login`)
      setSent(true)
    } catch (err) {
      setSendError(err instanceof Error ? err.message : 'Unknown error')
    } finally {
      setSending(false)
    }
  }

  if (verifying) {
    return <p className="p-6 text-center text-sm text-gray-500">Signing you in…</p>
  }

  if (sent) {
    return (
      <div className="p-6 text-center">
        <p className="text-brand-700">Check your email</p>
        <p className="mt-1 text-sm text-brand-500">We sent a sign-in link to {email}. It expires in 15 minutes.</p>
      </div>
    )
  }

  return (
    <div className="p-6">
      <h1 className="text-xl font-semibold text-brand-700">Sign in</h1>
      <p className="mt-1 text-sm text-brand-500">No password needed -- we'll email you a link.</p>
      {verifyError && <p className="mt-3 text-sm text-red-600">{verifyError}</p>}
      <form onSubmit={handleSubmit} className="mt-4 space-y-3">
        <div>
          <label htmlFor="login-email" className="mb-1 block text-xs font-medium text-brand-500">Email</label>
          <input
            id="login-email"
            type="email"
            required
            value={email}
            onChange={e => setEmail(e.target.value)}
            className="w-full rounded border border-gray-200 px-2 py-1.5 text-sm"
          />
        </div>
        {sendError && <p className="text-xs text-red-600">{sendError}</p>}
        <button
          type="submit"
          disabled={sending}
          className="rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-brand-700 disabled:opacity-50"
        >
          {sending ? 'Sending…' : 'Send sign-in link'}
        </button>
      </form>
    </div>
  )
}
