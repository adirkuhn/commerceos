import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { acceptInvite } from '../team/api'
import { AuthLayout, inputClass, labelClass, primaryButtonClass } from './AuthLayout'

export function AcceptInvite() {
  const { t } = useTranslation('auth')
  const token = new URLSearchParams(window.location.search).get('token')
  const [firstName, setFirstName] = useState('')
  const [lastName, setLastName] = useState('')
  const [password, setPassword] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [done, setDone] = useState(false)

  if (!token) {
    return <AuthLayout title="CommerceOS" subtitle={t('invite.noToken')} />
  }

  if (done) {
    return (
      <AuthLayout title={t('invite.doneTitle')} subtitle={t('invite.doneText')}>
        <a href="/" className={`${primaryButtonClass} block text-center`}>{t('invite.goToLogin')}</a>
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
      setError(err instanceof Error ? err.message : t('invite.unknownError'))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <AuthLayout
      title={t('invite.title')}
      subtitle={t('invite.subtitle')}
    >
      <form onSubmit={handleSubmit} className="space-y-3">
        <div>
          <label className={labelClass} htmlFor="ai-first-name">{t('invite.firstName')}</label>
          <input id="ai-first-name" autoComplete="given-name" className={inputClass} value={firstName} onChange={e => setFirstName(e.target.value)} />
        </div>
        <div>
          <label className={labelClass} htmlFor="ai-last-name">{t('invite.lastName')}</label>
          <input id="ai-last-name" autoComplete="family-name" className={inputClass} value={lastName} onChange={e => setLastName(e.target.value)} />
        </div>
        <div>
          <label className={labelClass} htmlFor="ai-password">{t('invite.password')}</label>
          <input id="ai-password" type="password" autoComplete="new-password" className={inputClass} value={password} onChange={e => setPassword(e.target.value)} />
        </div>

        {error && <p role="alert" className="text-xs text-danger">{error}</p>}

        <button type="submit" disabled={submitting} className={`${primaryButtonClass} !mt-5`}>
          {submitting ? t('invite.accepting') : t('invite.title')}
        </button>
      </form>
    </AuthLayout>
  )
}
