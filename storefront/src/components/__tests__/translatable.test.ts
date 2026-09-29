import { describe, it, expect } from 'vitest'
import { propsIn, translatableProps } from '../translatable'
import { registry } from '../registry'

describe('translatable props', () => {
  it('are a component\'s merchant texts: text, rich text and link lists', () => {
    expect(translatableProps(registry.get('hero')!)).toEqual(['title', 'subtitle', 'button_label'])
    expect(translatableProps(registry.get('navigation-menu')!)).toEqual(['items'])
    expect(translatableProps(registry.get('spacer')!)).toEqual([])
  })

  it('overlay the default props in one language', () => {
    const node = { id: 'h', type: 'heading', props: { text: 'Olá', level: 1 }, translations: { en: { text: 'Hello' } } }
    expect(propsIn(node, 'en')).toEqual({ text: 'Hello', level: 1 })
    expect(propsIn(node, 'nl')).toEqual({ text: 'Olá', level: 1 })
    expect(propsIn(node, undefined)).toEqual({ text: 'Olá', level: 1 })
  })
})
