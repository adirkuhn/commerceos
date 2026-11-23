import { Bot, ShieldAlert } from 'lucide-react'
import { useLocation } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useAuth } from '../auth/AuthContext'
import { idFromPath } from '../routes'
import { AssistantHome } from '../assistant/AssistantHome'
import { AssistantRequestView } from '../assistant/AssistantRequestView'
import { ASSISTANT_PATH, canUseAssistant } from '../assistant/format'

// The admin AI assistant: requests at /assistant, one conversation at
// /assistant/<id>. Admin roles only, like the API.
export default function Assistant() {
  const { t } = useTranslation('assistant')
  const { user } = useAuth()
  const { pathname } = useLocation()

  if (!canUseAssistant(user?.role)) {
    return (
      <div className="flex flex-col items-center justify-center h-full gap-2 text-ink-4">
        <ShieldAlert size={24} />
        <p className="text-sm">{t('adminOnly')}</p>
      </div>
    )
  }

  const id = idFromPath(pathname, ASSISTANT_PATH)
  return (
    <div className="flex flex-col h-full overflow-auto">
      <div className="flex items-center gap-2 px-5 py-3 border-b border-line bg-sunken shrink-0">
        <Bot size={14} className="text-accent" />
        <h1 className="text-sm font-semibold text-ink">{t('title')}</h1>
      </div>
      {id ? <AssistantRequestView key={id} id={id} /> : <AssistantHome />}
    </div>
  )
}
