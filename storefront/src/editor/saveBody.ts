import type { DocumentKind, ThemeNode } from '../core/document'
import type { PageTexts } from './inspector/PageTextsForm'

/** What saving the open document sends: its tree and its own texts. */
export function documentSaveBody(kind: DocumentKind, revision: number, root: ThemeNode, texts: PageTexts) {
  return {
    expected_revision: revision, root,
    ...(kind === 'page' ? { title: texts.title } : {}),
    seo: texts.seo,
    ...(texts.translations ? { translations: texts.translations } : {}),
  }
}
