import { describe, it, expect } from 'vitest'
import { documentIn } from '../document'

describe('documentIn', () => {
  const doc = {
    kind: 'page' as const, key: 'about', title: 'Sobre nós', seo: { title: 'Sobre', description: 'Quem somos' },
    translations: { en: { title: 'About us', seo: { title: 'About' } } }, root: { id: 'root', type: 'root' },
  }

  it('overlays a language’s texts, keeping the default where none was written', () => {
    expect(documentIn(doc, 'en')).toMatchObject({ title: 'About us', seo: { title: 'About', description: 'Quem somos' } })
    expect(documentIn(doc, 'nl')).toBe(doc)
    expect(documentIn(doc, undefined)).toBe(doc)
  })
})
