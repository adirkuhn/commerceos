import type { ComponentDefinition } from '../types'

export const SOCIAL_NETWORKS = ['whatsapp', 'instagram', 'facebook', 'tiktok', 'youtube', 'x', 'maps', 'email', 'phone'] as const
export type SocialNetwork = (typeof SOCIAL_NETWORKS)[number]

export interface SocialLink { network: SocialNetwork; link: string }
export interface SocialLinksProps { items: SocialLink[]; show_labels: boolean; size: 'sm' | 'md' | 'lg' }

export const socialLinks: ComponentDefinition<SocialLinksProps> = {
  type: 'social-links', name: 'Social links', category: 'navigation', icon: 'share',
  description: 'Icons linking to your WhatsApp, Instagram, Facebook, map location and more.',
  props: {
    items: { kind: 'socialLinks', label: 'Links', max: 8 },
    show_labels: { kind: 'toggle', label: 'Show names next to icons' },
    size: { kind: 'select', label: 'Icon size', options: [{ value: 'sm', label: 'Small' }, { value: 'md', label: 'Medium' }, { value: 'lg', label: 'Large' }] },
  },
  defaults: { items: [{ network: 'instagram', link: 'https://instagram.com/' }], show_labels: false, size: 'md' },
  container: false, allowedChildren: [], allowedParents: [], dataContext: '', requiredIn: [],
  editor: { removable: true, copyable: true, draggable: true },
  styleGroups: ['spacing', 'layout', 'visibility'],
}
