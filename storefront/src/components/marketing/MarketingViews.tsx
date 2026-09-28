import { useState } from 'react'
import { useSuspenseQuery } from '@tanstack/react-query'
import { useStorefrontAPI } from '../../api/ApiContext'
import { resolveAssetRef, useRenderEnv } from '../../runtime/RenderContext'
import { storeHref } from '../../runtime/page/links'
import { collectionsQuery } from '../../runtime/page/queries'
import type { RendererProps } from '../types'
import type { BannerProps, CollectionListProps, HeroProps, NewsletterSignupProps } from './marketing'

export function HeroView({ node, props, className }: RendererProps<HeroProps>) {
  const env = useRenderEnv()
  const image = resolveAssetRef(props.image, env.assets)
  const href = storeHref(props.button_link, env.storePath)
  const height = ['md', 'lg', 'screen'].includes(props.height) ? props.height : 'lg'
  return (
    <div className={`cos-hero cos-hero-${height} cos-hero-${props.align === 'left' ? 'left' : 'center'} ${className}`} data-node={node.id}
      style={image ? { backgroundImage: `url("${encodeURI(image.url).replace(/"/g, '%22')}")` } : undefined}>
      <div className="cos-hero-content">
        <h1 className="cos-heading">{props.title}</h1>
        {props.subtitle && <p>{props.subtitle}</p>}
        {props.button_label && href && (
          <a className="cos-button cos-button-primary" href={env.mode === 'live' ? href : undefined}>{props.button_label}</a>
        )}
      </div>
    </div>
  )
}

export function BannerView({ node, props, className }: RendererProps<BannerProps>) {
  const env = useRenderEnv()
  const href = storeHref(props.link, env.storePath)
  return (
    <div className={`cos-banner ${className}`} data-node={node.id} role="note">
      <span>{props.text}</span>
      {props.link_label && href && <a href={env.mode === 'live' ? href : undefined}>{props.link_label}</a>}
    </div>
  )
}

export function CollectionListView({ node, props, className }: RendererProps<CollectionListProps>) {
  const api = useStorefrontAPI()
  const env = useRenderEnv()
  const { data } = useSuspenseQuery(collectionsQuery(api))
  const picked = props.source === 'picked'
    ? (props.collection_ids ?? []).map(id => data.find(c => c.id === id)).filter(c => c !== undefined)
    : data.slice(0, Math.min(Math.max(Number(props.limit) || 6, 1), 24))
  if (!picked.length) {
    return env.mode === 'editor' ? <div className={`cos-placeholder ${className}`} data-node={node.id}>Publish a collection to show it here.</div> : null
  }
  return (
    <ul className={`cos-collection-list ${className}`} data-node={node.id}>
      {picked.map(c => (
        <li key={c.id}>
          <a href={env.mode === 'live' ? env.storePath(`collections/${encodeURIComponent(c.slug)}`) : undefined} className="cos-collection-tile">
            <div className="cos-collection-tile-image">{c.image_url && <img src={c.image_url} alt="" loading="lazy" />}</div>
            <span>{c.name}</span>
          </a>
        </li>
      ))}
    </ul>
  )
}

export function NewsletterSignupView({ node, props, className }: RendererProps<NewsletterSignupProps>) {
  const api = useStorefrontAPI()
  const env = useRenderEnv()
  const [email, setEmail] = useState('')
  const [consent, setConsent] = useState(false)
  const [state, setState] = useState<'idle' | 'busy' | 'done' | 'error'>('idle')
  const [error, setError] = useState('')
  const id = `${node.id}-newsletter`

  const submit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (env.mode === 'editor') return
    setState('busy')
    try {
      await api.subscribeNewsletter(email, consent)
      setState('done')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Something went wrong')
      setState('error')
    }
  }

  return (
    <section className={`cos-newsletter ${className}`} data-node={node.id} aria-labelledby={props.title ? `${id}-title` : undefined}>
      {props.title && <h2 id={`${id}-title`} className="cos-heading">{props.title}</h2>}
      {props.text && <p>{props.text}</p>}
      {state === 'done' ? (
        <p role="status">Almost done: check your inbox and confirm your subscription.</p>
      ) : (
        <form onSubmit={e => void submit(e)}>
          <label htmlFor={`${id}-email`} className="cos-label">Email</label>
          <div className="cos-newsletter-row">
            <input id={`${id}-email`} type="email" required autoComplete="email" value={email} onChange={e => setEmail(e.target.value)} />
            <button type="submit" className="cos-button cos-button-primary" disabled={state === 'busy' || !consent}>{props.button_label || 'Subscribe'}</button>
          </div>
          <label className="cos-consent">
            <input type="checkbox" required checked={consent} onChange={e => setConsent(e.target.checked)} /> {props.consent_label}
          </label>
          {state === 'error' && <p role="alert" className="cos-error">{error}</p>}
        </form>
      )}
    </section>
  )
}
