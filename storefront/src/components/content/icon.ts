import type { ComponentDefinition } from '../types'

export const ICON_NAMES = ['refresh', 'box', 'gift', 'clock', 'truck', 'leaf', 'heart', 'star', 'shield', 'phone', 'tag', 'smile'] as const
export type IconName = (typeof ICON_NAMES)[number]

export interface IconProps { name: IconName; size: 'sm' | 'md' | 'lg' }

const iconLabels: Record<IconName, string> = {
  refresh: 'Returns', box: 'Box', gift: 'Gift', clock: 'Clock', truck: 'Delivery truck', leaf: 'Leaf',
  heart: 'Heart', star: 'Star', shield: 'Shield', phone: 'Phone', tag: 'Price tag', smile: 'Smile',
}

export const icon: ComponentDefinition<IconProps> = {
  type: 'icon', name: 'Icon', category: 'content', icon: 'star',
  description: 'A simple line icon in your primary colour, e.g. above a service promise.',
  props: {
    name: { kind: 'select', label: 'Icon', options: ICON_NAMES.map(n => ({ value: n, label: iconLabels[n] })) },
    size: { kind: 'select', label: 'Size', options: [{ value: 'sm', label: 'Small' }, { value: 'md', label: 'Medium' }, { value: 'lg', label: 'Large' }] },
  },
  defaults: { name: 'star', size: 'md' },
  container: false, allowedChildren: [], allowedParents: [], dataContext: '', requiredIn: [],
  editor: { removable: true, copyable: true, draggable: true },
  styleGroups: ['spacing', 'layout', 'visibility'],
}
