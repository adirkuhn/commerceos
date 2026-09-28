import { safeHref } from '../../core/richText'
import { useRenderEnv } from '../../runtime/RenderContext'
import type { RendererProps } from '../types'
import type { ButtonProps } from './button'

const VARIANTS = new Set(['primary', 'secondary', 'outline'])

export function ButtonView({ node, props, className }: RendererProps<ButtonProps>) {
  const env = useRenderEnv()
  const variant = VARIANTS.has(props.variant) ? props.variant : 'primary'
  const href = safeHref(props.link) ?? '#'
  const live = env.mode === 'live'
  return (
    <a className={`cos-button cos-button-${variant} ${className}`} data-node={node.id}
      href={live ? href : undefined} role={live ? undefined : 'link'} aria-disabled={live ? undefined : true}
      {...(live && props.new_tab ? { target: '_blank', rel: 'noopener noreferrer' } : {})}>
      {props.label}
    </a>
  )
}
