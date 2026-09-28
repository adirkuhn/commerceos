import { useEffect, useState } from 'react'
import { useStorefrontAPI } from '../../api/ApiContext'
import { useRenderEnv } from '../RenderContext'

type Action = 'confirm' | 'unsubscribe'
type State = 'working' | 'done' | 'failed'

const COPY: Record<Action, Record<State, string>> = {
  confirm: {
    working: 'Confirming your subscription…',
    done: 'You’re subscribed. Thanks for joining our newsletter!',
    failed: 'This confirmation link has expired or was already used. Sign up again to get a new one.',
  },
  unsubscribe: {
    working: 'Unsubscribing…',
    done: 'You’ve been unsubscribed. You won’t get our newsletter anymore.',
    failed: 'This unsubscribe link is not valid. If you still get our emails, use the link in the latest one.',
  },
}

/** Where the newsletter emails' links land. The token only runs in the
 *  browser (never on the server), so link scanners can't act on it. */
export function NewsletterPage({ action }: { action: Action }) {
  const api = useStorefrontAPI()
  const env = useRenderEnv()
  const [state, setState] = useState<State>('working')

  useEffect(() => {
    const token = new URLSearchParams(window.location.search).get('token')
    if (!token) {
      setState('failed')
      return
    }
    const call = action === 'confirm' ? api.confirmNewsletter(token) : api.unsubscribeNewsletter(token)
    call.then(() => setState('done'), () => setState('failed'))
  }, [action, api])

  return (
    <section className="cos-newsletter-page" aria-live="polite">
      <h1 className="cos-heading">Newsletter</h1>
      <p role={state === 'failed' ? 'alert' : undefined}>{COPY[action][state]}</p>
      {state !== 'working' && <a className="cos-button cos-button-outline" href={env.storePath('')}>Continue shopping</a>}
    </section>
  )
}
