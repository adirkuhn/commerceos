import type { ComponentDefinition } from '../types'

export interface SpacerProps {
  size: 'xs' | 'sm' | 'md' | 'lg' | 'xl' | '2xl' | '3xl'
}

export const spacer: ComponentDefinition<SpacerProps> = {
  type: 'spacer',
  name: 'Spacer',
  category: 'layout',
  icon: 'spacer',
  description: 'Empty vertical space between components.',
  props: {
    size: { kind: 'select', label: 'Height', options: ['xs', 'sm', 'md', 'lg', 'xl', '2xl', '3xl'].map(v => ({ value: v, label: v.toUpperCase() })) },
  },
  defaults: { size: 'lg' },
  container: false,
  allowedChildren: [],
  allowedParents: [],
  dataContext: '',
  requiredIn: [],
  styleGroups: ['visibility'],
  editor: { removable: true, copyable: true, draggable: true },
}
