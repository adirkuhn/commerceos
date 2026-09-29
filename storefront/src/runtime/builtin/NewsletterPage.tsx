import { useEffect, useState } from 'react'
import type { TFunction } from 'i18next'
import { useTranslation } from 'react-i18next'
import { useStorefrontAPI } from '../../api/ApiContext'
import { useRenderEnv } from '../RenderContext'

type Action = 'confirm' | 'unsubscribe'
type State = 'working' | 'done' | 'failed'

function copy(t: TFunction): Record<Action, Record<State, string>> {
  return {
    confirm: { working: t('newsletter.confirming'), done: t('newsletter.confirmed'), failed: t('newsletter.confirmFailed') },
    unsubscribe: { working: t('newsletter.unsubscribing'), done: t('newsletter.unsubscribed'), failed: t('newsletter.unsubscribeFailed') },
  }
}

/** Where the newsletter emails' links land. The token only runs in the
 *  browser (never on the server), so link scanners can't act on it. */
export function NewsletterPage({ action }: { action: Action }) {
  const api = useStorefrontAPI()
  const env = useRenderEnv()
  const { t } = useTranslation()
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
      <h1 className="cos-heading">{t('newsletter.title')}</h1>
      <p role={state === 'failed' ? 'alert' : undefined}>{copy(t)[action][state]}</p>
      {state !== 'working' && <a className="cos-button cos-button-outline" href={env.storePath('')}>{t('common.continueShopping')}</a>}
    </section>
  )
}
