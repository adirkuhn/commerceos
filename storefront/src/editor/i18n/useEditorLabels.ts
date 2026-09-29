import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { createEditorLabels, type EditorLabels } from './labels'

export function useEditorLabels(): EditorLabels {
  const { t } = useTranslation('editor')
  return useMemo(() => createEditorLabels(t), [t])
}
