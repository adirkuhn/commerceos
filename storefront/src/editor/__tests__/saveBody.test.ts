import { describe, it, expect } from 'vitest'
import { documentSaveBody } from '../saveBody'

const root = { id: 'root', type: 'root' }

describe('documentSaveBody', () => {
  // Saving used to send only the tree: a page could not be saved (its
  // title is required) and a template lost its search-engine texts.
  it('sends a page\'s title, search texts and translations with the tree', () => {
    expect(documentSaveBody('page', 3, root, { title: 'Sobre', seo: { title: 'Sobre' }, translations: { en: { title: 'About' } } })).toEqual({
      expected_revision: 3, root, title: 'Sobre', seo: { title: 'Sobre' }, translations: { en: { title: 'About' } },
    })
  })

  it('templates send no title', () => {
    expect(documentSaveBody('template', 1, root, { title: 'Home', seo: {} })).toEqual({ expected_revision: 1, root, seo: {} })
  })
})
