import type { ComponentDefinition } from '../types'

export interface ImageProps {
  image: string
  alt: string
  link: string
  fit: 'cover' | 'contain'
}

export const image: ComponentDefinition<ImageProps> = {
  type: 'image',
  name: 'Image',
  category: 'content',
  icon: 'image',
  description: 'A picture from your media library, optionally linked.',
  props: {
    image: { kind: 'asset', label: 'Image' },
    alt: { kind: 'text', label: 'Description (for screen readers)', maxLength: 250 },
    link: { kind: 'link', label: 'Link' },
    fit: { kind: 'select', label: 'Fit', options: [{ value: 'cover', label: 'Fill the frame' }, { value: 'contain', label: 'Show whole image' }] },
  },
  defaults: { image: '', alt: '', link: '', fit: 'cover' },
  container: false,
  allowedChildren: [],
  allowedParents: [],
  dataContext: '',
  requiredIn: [],
  styleGroups: ['spacing', 'border', 'layout', 'visibility'],
  editor: { removable: true, copyable: true, draggable: true },
}
