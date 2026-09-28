import type { ComponentDefinition } from '../types'

const base = {
  container: false, allowedChildren: [], allowedParents: [], dataContext: '' as const, requiredIn: [],
  editor: { removable: true, copyable: true, draggable: true },
}

export interface HeroProps {
  title: string; subtitle: string; image: string; button_label: string; button_link: string
  align: 'left' | 'center'; height: 'md' | 'lg' | 'screen'
}
export const hero: ComponentDefinition<HeroProps> = {
  ...base, type: 'hero', name: 'Hero', category: 'content', icon: 'image',
  description: 'A big welcome: picture, title, text and a button.',
  props: {
    title: { kind: 'text', label: 'Title', maxLength: 120 },
    subtitle: { kind: 'text', label: 'Text', maxLength: 300, multiline: true },
    image: { kind: 'asset', label: 'Background picture' },
    button_label: { kind: 'text', label: 'Button label', maxLength: 60 },
    button_link: { kind: 'link', label: 'Button link' },
    align: { kind: 'select', label: 'Alignment', options: [{ value: 'center', label: 'Centred' }, { value: 'left', label: 'Left' }] },
    height: { kind: 'select', label: 'Height', options: [{ value: 'md', label: 'Medium' }, { value: 'lg', label: 'Tall' }, { value: 'screen', label: 'Full screen' }] },
  },
  defaults: { title: 'Welcome', subtitle: '', image: '', button_label: 'Shop now', button_link: '/', align: 'center', height: 'lg' },
  styleGroups: ['colors', 'spacing', 'visibility'],
}

export interface BannerProps { text: string; link_label: string; link: string }
export const banner: ComponentDefinition<BannerProps> = {
  ...base, type: 'banner', name: 'Banner', category: 'content', icon: 'megaphone',
  description: 'A strip for an announcement or promotion.',
  props: {
    text: { kind: 'text', label: 'Text', maxLength: 200 },
    link_label: { kind: 'text', label: 'Link label', maxLength: 60 },
    link: { kind: 'link', label: 'Link' },
  },
  defaults: { text: 'Free shipping on all orders', link_label: '', link: '' },
  styleGroups: ['colors', 'typography', 'spacing', 'visibility'],
}

export interface CollectionListProps { source: 'all' | 'picked'; collection_ids: string[]; limit: number }
export const collectionList: ComponentDefinition<CollectionListProps> = {
  ...base, type: 'collection-list', name: 'Collection list', category: 'commerce', icon: 'layers',
  description: 'Tiles linking to your collections.',
  props: {
    source: { kind: 'select', label: 'Show', options: [{ value: 'all', label: 'All collections' }, { value: 'picked', label: 'Collections I choose' }] },
    collection_ids: { kind: 'collections', label: 'Collections', max: 24 },
    limit: { kind: 'number', label: 'How many', min: 1, max: 24 },
  },
  defaults: { source: 'all', collection_ids: [], limit: 6 },
  showField: { collection_ids: p => p.source === 'picked', limit: p => p.source === 'all' },
  styleGroups: ['spacing', 'visibility'],
}

export interface NewsletterSignupProps { title: string; text: string; button_label: string; consent_label: string }
export const newsletterSignup: ComponentDefinition<NewsletterSignupProps> = {
  ...base, type: 'newsletter-signup', name: 'Newsletter signup', category: 'content', icon: 'mail',
  description: 'Email signup with consent. Shoppers confirm by email before they are subscribed.',
  props: {
    title: { kind: 'text', label: 'Title', maxLength: 120 },
    text: { kind: 'text', label: 'Text', maxLength: 300 },
    button_label: { kind: 'text', label: 'Button label', maxLength: 40 },
    consent_label: { kind: 'text', label: 'Consent text', maxLength: 200, multiline: true },
  },
  defaults: { title: 'Stay in the loop', text: '', button_label: 'Subscribe', consent_label: 'I would like to receive emails from this store. I can unsubscribe at any time.' },
  styleGroups: ['colors', 'spacing', 'visibility'],
}
