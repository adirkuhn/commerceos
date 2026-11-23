import { useTranslation } from 'react-i18next'
import { useConfirm } from '../components/common/useConfirm'
import type { AssistantActions } from './hooks/useAssistantActions'
import type { AssistantRequest } from './types'

// Confirm, then cancel -- shared by the list rows and the conversation.
export function useCancelRequest(actions: AssistantActions, onCancelled: (r: AssistantRequest) => void) {
  const { t } = useTranslation('assistant')
  const { confirm, confirmDialog } = useConfirm()

  async function cancel(id: string) {
    const ok = await confirm({ title: t('cancel.title'), message: t('cancel.message'), confirmLabel: t('cancel.confirm'), destructive: true })
    if (!ok) return
    const out = await actions.cancel(id)
    if (out) onCancelled(out)
  }

  return { cancel, confirmDialog }
}
