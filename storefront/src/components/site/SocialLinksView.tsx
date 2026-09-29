import { useTranslation } from 'react-i18next'
import { safeHref } from '../../core/richText'
import { useRenderEnv } from '../../runtime/RenderContext'
import type { RendererProps } from '../types'
import { SOCIAL_NETWORKS, type SocialNetwork, type SocialLinksProps } from './social'
import { SocialIcon } from './SocialIcon'

const ICON_SIZE = { sm: 18, md: 22, lg: 28 } as const

export function SocialLinksView({ node, props, className }: RendererProps<SocialLinksProps>) {
  const env = useRenderEnv()
  const { t } = useTranslation()
  const names: Record<SocialNetwork, string> = {
    whatsapp: 'WhatsApp', instagram: 'Instagram', facebook: 'Facebook', tiktok: 'TikTok', youtube: 'YouTube', x: 'X',
    maps: t('social.maps'), email: t('social.email'), phone: t('social.phone'),
  }
  const items = (props.items ?? []).filter(i => SOCIAL_NETWORKS.includes(i.network) && safeHref(i.link)).slice(0, 8)
  const size = ICON_SIZE[props.size] ?? ICON_SIZE.md
  if (!items.length) return env.mode === 'editor' ? <div className={`cos-placeholder ${className}`} data-node={node.id}>{t('hint.socialLinks')}</div> : null
  return (
    <ul className={`cos-social ${className}`} data-node={node.id} aria-label={t('social.label')}>
      {items.map((item, i) => {
        const external = /^https?:/.test(item.link)
        return (
          <li key={i}>
            <a href={env.mode === 'live' ? item.link : undefined} aria-label={props.show_labels ? undefined : names[item.network]}
              {...(external ? { target: '_blank', rel: 'noopener noreferrer' } : {})}>
              <SocialIcon network={item.network} size={size} />
              {props.show_labels && <span>{names[item.network]}</span>}
            </a>
          </li>
        )
      })}
    </ul>
  )
}
