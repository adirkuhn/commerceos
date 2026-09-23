import { useState } from 'react'
import { acceptInvite } from '../team/api'
import { AuthLayout, inputClass, labelClass, primaryButtonClass } from './AuthLayout'

export function AcceptInvite() {
  const token = new URLSearchParams(window.location.search).get('token')
  const [firstName, setFirstName] = useState('')
  const [lastName, setLastName] = useState('')
  const [password, setPassword] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [done, setDone] = useState(false)

  if (!token) {
    return <AuthLayout title="CommerceOS" subtitle="No invite token found in the URL." />
  }

  if (done) {
    return (
      <AuthLayout title="You're in!" subtitle="Your invite has been accepted. Sign in to continue.">
        <a href="/" className={`${primaryButtonClass} block text-center`}>Go to login</a>
      </AuthLayout>
    )
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setSubmitting(true)
    setError(null)
    try {
      await acceptInvite({ token, firstName, lastName, password })
      setDone(true)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <AuthLayout
      title="Accept Invite"
      subtitle="Already have an Akantio account? Just click Accept. New here? Fill in your name and pick a password."
    >
      <form onSubmit={handleSubmit} className="space-y-3">
        <div>
          <label className={labelClass} htmlFor="ai-first-name">First name</label>
          <input id="ai-first-name" autoComplete="given-name" className={inputClass} value={firstName} onChange={e => setFirstName(e.target.value)} />
        </div>
        <div>
          <label className={labelClass} htmlFor="ai-last-name">Last name</label>
          <input id="ai-last-name" autoComplete="family-name" className={inputClass} value={lastName} onChange={e => setLastName(e.target.value)} />
        </div>
        <div>
          <label className={labelClass} htmlFor="ai-password">Password (new account only)</label>
          <input id="ai-password" type="password" autoComplete="new-password" className={inputClass} value={password} onChange={e => setPassword(e.target.value)} />
        </div>

        {error && <p role="alert" className="text-xs text-danger">{error}</p>}

        <button type="submit" disabled={submitting} className={`${primaryButtonClass} !mt-5`}>
          {submitting ? 'Accepting…' : 'Accept Invite'}
        </button>
      </form>
    </AuthLayout>
  )
}
