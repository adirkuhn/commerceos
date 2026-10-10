import { describe, it, expect } from 'vitest'
import { createEditorI18n } from '../i18n/createEditorI18n'
import { editorErrorText } from '../editorErrorText'
import { EditorApiError } from '../api'

describe('editorErrorText', () => {
  it('shows a theme error code in the editor language', () => {
    const nl = createEditorI18n('nl')
    expect(editorErrorText(new EditorApiError(422, 'THEME_ASSET_TOO_LARGE', 'images can be at most 10 MB'), nl, 'fallback'))
      .toBe('Afbeeldingen mogen maximaal 10 MB zijn.')
  })

  it('keeps a page-validation detail and plain errors as they are', () => {
    const pt = createEditorI18n('pt-BR')
    expect(editorErrorText(new EditorApiError(422, 'VALIDATION_ERROR', 'heading: text too long', 'n1'), pt, 'fallback')).toBe('heading: text too long')
    expect(editorErrorText(new Error('network'), pt, 'fallback')).toBe('network')
    expect(editorErrorText('x', pt, 'fallback')).toBe('fallback')
  })
})
