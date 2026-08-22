import { useState } from 'react'
import { acceptInvite } from '../team/api'

const shellStyle: React.CSSProperties = {
  display: 'flex', alignItems: 'center', justifyContent: 'center', minHeight: '100vh', background: '#f5f5f5',
}
const cardStyle: React.CSSProperties = {
  background: '#fff', padding: '48px', borderRadius: '8px', boxShadow: '0 2px 8px rgba(0,0,0,0.1)', minWidth: '320px', maxWidth: '380px',
}
const inputStyle: React.CSSProperties = {
  width: '100%', padding: '10px', marginBottom: '12px', borderRadius: '6px', border: '1px solid #ddd', fontSize: '0.95rem',
}
const labelStyle: React.CSSProperties = { display: 'block', marginBottom: '4px', fontSize: '0.85rem', color: '#555' }

export function AcceptInvite() {
  const token = new URLSearchParams(window.location.search).get('token')
  const [firstName, setFirstName] = useState('')
  const [lastName, setLastName] = useState('')
  const [password, setPassword] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [done, setDone] = useState(false)

  if (!token) {
    return (
      <div style={shellStyle}>
        <div style={cardStyle}>
          <h1 style={{ margin: '0 0 8px', fontSize: '1.5rem' }}>CommerceOS</h1>
          <p style={{ margin: 0, color: '#666' }}>No invite token found in the URL.</p>
        </div>
      </div>
    )
  }

  if (done) {
    return (
      <div style={shellStyle}>
        <div style={cardStyle}>
          <h1 style={{ margin: '0 0 8px', fontSize: '1.5rem' }}>You&apos;re in!</h1>
          <p style={{ margin: '0 0 24px', color: '#666' }}>Your invite has been accepted. Sign in to continue.</p>
          <a href="/">Go to login</a>
        </div>
      </div>
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
    <div style={shellStyle}>
      <div style={cardStyle}>
        <h1 style={{ margin: '0 0 8px', fontSize: '1.5rem' }}>Accept Invite</h1>
        <p style={{ margin: '0 0 24px', color: '#666' }}>
          Already have an Akantio account? Just click Accept. New here? Fill in your name and pick a password.
        </p>
        <form onSubmit={handleSubmit}>
          <label style={labelStyle} htmlFor="ai-first-name">First name</label>
          <input id="ai-first-name" style={inputStyle} value={firstName} onChange={e => setFirstName(e.target.value)} />

          <label style={labelStyle} htmlFor="ai-last-name">Last name</label>
          <input id="ai-last-name" style={inputStyle} value={lastName} onChange={e => setLastName(e.target.value)} />

          <label style={labelStyle} htmlFor="ai-password">Password (new account only)</label>
          <input id="ai-password" type="password" style={inputStyle} value={password} onChange={e => setPassword(e.target.value)} />

          {error && <p style={{ color: 'red', fontSize: '0.85rem', margin: '0 0 12px' }}>{error}</p>}

          <button
            type="submit"
            disabled={submitting}
            style={{
              width: '100%', padding: '12px', background: '#1a1a1a', color: '#fff', border: 'none',
              borderRadius: '6px', fontSize: '1rem', cursor: 'pointer',
            }}
          >
            {submitting ? 'Accepting…' : 'Accept Invite'}
          </button>
        </form>
      </div>
    </div>
  )
}
