import { useState } from 'react'
import clsx from 'clsx'
import MessageTemplatesPanel from '../messagecenter/MessageTemplatesPanel'
import AutomationRulesPanel from '../messagecenter/AutomationRulesPanel'

type Tab = 'templates' | 'rules'

export default function MessageCenter() {
  const [tab, setTab] = useState<Tab>('templates')

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center gap-1 px-5 pt-3 border-b dark:border-[#2D3748] border-gray-200 dark:bg-[#111827] bg-gray-50 shrink-0">
        <button
          onClick={() => setTab('templates')}
          className={clsx(
            'px-3 py-2 text-xs font-medium border-b-2 transition-colors',
            tab === 'templates'
              ? 'border-blue-500 dark:text-gray-100 text-gray-900'
              : 'border-transparent dark:text-gray-500 text-gray-400 hover:dark:text-gray-300 hover:text-gray-600',
          )}
        >
          Templates
        </button>
        <button
          onClick={() => setTab('rules')}
          className={clsx(
            'px-3 py-2 text-xs font-medium border-b-2 transition-colors',
            tab === 'rules'
              ? 'border-blue-500 dark:text-gray-100 text-gray-900'
              : 'border-transparent dark:text-gray-500 text-gray-400 hover:dark:text-gray-300 hover:text-gray-600',
          )}
        >
          Automation Rules
        </button>
      </div>

      <div className="flex-1 min-h-0">
        {tab === 'templates' ? <MessageTemplatesPanel /> : <AutomationRulesPanel />}
      </div>
    </div>
  )
}
