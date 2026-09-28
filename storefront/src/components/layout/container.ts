import type { ComponentDefinition } from '../types'

export type ContainerProps = Record<string, never>

export const container: ComponentDefinition<ContainerProps> = {
  type: 'container',
  name: 'Container',
  category: 'layout',
  icon: 'square',
  description: 'Groups components and stacks them with even spacing.',
  props: {},
  defaults: {},
  container: true,
  element: () => ({ tag: 'div', className: 'cos-container' }),
  allowedChildren: [],
  allowedParents: [],
  dataContext: '',
  requiredIn: [],
  styleGroups: ['spacing', 'colors', 'background', 'border', 'layout', 'visibility'],
  editor: { removable: true, copyable: true, draggable: true },
}
