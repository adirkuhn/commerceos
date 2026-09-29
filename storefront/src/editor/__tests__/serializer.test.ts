import { describe, it, expect } from 'vitest'
import { fromGrapes, toGrapes, type GrapesComponentLike, type GrapesDefinition } from '../grapes/serializer'
import { registry } from '../../components/registry'
import type { ThemeNode } from '../../core/document'

// A minimal stand-in for a GrapesJS component model.
function fake(def: GrapesDefinition): GrapesComponentLike {
  const children = (def.components ?? []).map(fake)
  return {
    get: (k: string) => (def as unknown as Record<string, unknown>)[k],
    components: () => children,
  }
}

const tree: ThemeNode = {
  id: 'root', type: 'root', children: [{
    id: 's1', type: 'section', props: { full_width: false }, style: { base: { padding_y: 'xl' } },
    children: [{ id: 'h1', type: 'heading', props: { text: 'Hi', level: 1 }, style: { mobile: { font_size: 'xl' } } }],
  }],
}

describe('serializer', () => {
  it('builds GrapesJS definitions with container markup from the registry', () => {
    const [section] = toGrapes(tree, registry)
    expect(section).toMatchObject({ type: 'section', tagName: 'section', classes: ['cos-section', 'cos-contained', 'n-s1'], cosId: 's1' })
    expect(section.components![0]).toMatchObject({ type: 'heading', cosId: 'h1', cosProps: { text: 'Hi', level: 1 } })
  })

  it('round-trips a document', () => {
    const wrapper: GrapesComponentLike = { get: () => undefined, components: () => toGrapes(tree, registry).map(fake) }
    expect(fromGrapes(wrapper)).toEqual(tree)
  })

  it('gives duplicated (copied) nodes fresh unique ids', () => {
    const [section] = toGrapes(tree, registry)
    const dup = { ...section.components![0] }
    section.components!.push(dup)
    const wrapper: GrapesComponentLike = { get: () => undefined, components: () => [fake(section)] }

    const out = fromGrapes(wrapper, () => 'fresh1')
    const ids = out.children![0].children!.map(n => n.id)
    expect(ids).toEqual(['h1', 'fresh1'])
  })

  it('drops empty props and styles', () => {
    const wrapper: GrapesComponentLike = {
      get: () => undefined,
      components: () => [fake({ type: 'container', cosId: 'c', cosProps: {}, cosStyle: {} })],
    }
    expect(fromGrapes(wrapper).children![0]).toEqual({ id: 'c', type: 'container' })
  })
})

describe('serializer translations', () => {
  const translated: ThemeNode = { id: 'root', type: 'root', children: [
    { id: 'h1', type: 'heading', props: { text: 'Olá', level: 1 }, translations: { en: { text: 'Hello' } } },
  ] }

  it('round-trips a node\'s translations', () => {
    const [h] = toGrapes(translated, registry)
    expect(h.cosTranslations).toEqual({ en: { text: 'Hello' } })
    const wrapper: GrapesComponentLike = { get: () => undefined, components: () => toGrapes(translated, registry).map(fake) }
    expect(fromGrapes(wrapper)).toEqual(translated)
  })

  it('drops blank translations, so the default text shows', () => {
    const [h] = toGrapes(translated, registry)
    h.cosTranslations = { en: { text: '' }, nl: { text: 'Hallo' } }
    const wrapper: GrapesComponentLike = { get: () => undefined, components: () => [fake(h)] }
    expect(fromGrapes(wrapper).children![0].translations).toEqual({ nl: { text: 'Hallo' } })
    h.cosTranslations = { en: { text: '' } }
    expect(fromGrapes({ get: () => undefined, components: () => [fake(h)] }).children![0].translations).toBeUndefined()
  })
})
