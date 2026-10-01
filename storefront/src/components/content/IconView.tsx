import type { ReactNode } from 'react'
import type { RendererProps } from '../types'
import { ICON_NAMES, type IconName, type IconProps } from './icon'

// 24x24 line drawings, stroked in the current colour.
const SHAPES: Record<IconName, ReactNode> = {
  refresh: <><path d="M20 11a8 8 0 0 0-14.3-4.9L4 8" /><path d="M4 4v4h4" /><path d="M4 13a8 8 0 0 0 14.3 4.9L20 16" /><path d="M20 20v-4h-4" /></>,
  box: <><path d="M12 3 3.5 7.5v9L12 21l8.5-4.5v-9z" /><path d="M3.5 7.5 12 12l8.5-4.5" /><path d="M12 12v9" /></>,
  gift: <><rect x="3.5" y="8" width="17" height="4" /><path d="M5 12v9h14v-9" /><path d="M12 8v13" /><path d="M12 8C10 4 6.5 4.5 7 6.5S12 8 12 8s4.5 0 5-1.5S14 4 12 8z" /></>,
  clock: <><circle cx="12" cy="12" r="9" /><path d="M12 7v5l3 2" /></>,
  truck: <><path d="M2.5 6h11v10h-11z" /><path d="M13.5 9.5h4l3 3.5v3h-7z" /><circle cx="6.5" cy="17.5" r="1.8" /><circle cx="17" cy="17.5" r="1.8" /></>,
  leaf: <><path d="M5 19C5 10 10 5 20 4c-1 10-6 15-15 15z" /><path d="M5 19 13 11" /></>,
  heart: <path d="M12 20s-7.5-4.6-7.5-10.2A4.3 4.3 0 0 1 12 7a4.3 4.3 0 0 1 7.5 2.8C19.5 15.4 12 20 12 20z" />,
  star: <path d="m12 3.5 2.6 5.3 5.9.9-4.3 4.1 1 5.8-5.2-2.7-5.2 2.7 1-5.8-4.3-4.1 5.9-.9z" />,
  shield: <><path d="M12 3 5 6v6c0 4.5 3 7.8 7 9 4-1.2 7-4.5 7-9V6z" /><path d="m9 12 2 2 4-4" /></>,
  phone: <path d="M6.6 3.5h3l1.5 4-2 1.3a11 11 0 0 0 6.1 6.1l1.3-2 4 1.5v3a2 2 0 0 1-2.2 2A17 17 0 0 1 4.6 5.7a2 2 0 0 1 2-2.2z" />,
  tag: <><path d="M3.5 12.5V4h8.5l8.5 8.5-8.5 8.5z" /><circle cx="8" cy="8.5" r="1.3" /></>,
  smile: <><circle cx="12" cy="12" r="9" /><path d="M8.5 14.5a4.5 4.5 0 0 0 7 0" /><path d="M9 9.5h.01M15 9.5h.01" /></>,
}

export function IconView({ node, props, className }: RendererProps<IconProps>) {
  const name = ICON_NAMES.includes(props.name) ? props.name : 'star'
  const size = ['sm', 'md', 'lg'].includes(props.size) ? props.size : 'md'
  return (
    <span className={`cos-icon cos-icon-${size} ${className}`} data-node={node.id} aria-hidden="true">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.6} strokeLinecap="round" strokeLinejoin="round">{SHAPES[name]}</svg>
    </span>
  )
}
