import type { ComponentDefinition } from '../types'
import type { RichText } from '../../core/richText'

export interface TextProps {
  content: RichText
}

export const text: ComponentDefinition<TextProps> = {
  type: 'text',
  name: 'Text',
  category: 'content',
  icon: 'text',
  description: 'Paragraphs and lists, with bold, italic and links.',
  props: { content: { kind: 'richtext', label: 'Text' } },
  defaults: { content: [{ type: 'paragraph', children: [{ text: 'Write something here.' }] }] },
  container: false,
  allowedChildren: [],
  allowedParents: [],
  dataContext: '',
  requiredIn: [],
  styleGroups: ['typography', 'colors', 'spacing', 'visibility'],
  editor: { removable: true, copyable: true, draggable: true },
}
