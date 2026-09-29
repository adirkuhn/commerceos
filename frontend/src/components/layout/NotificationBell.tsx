import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Bell } from 'lucide-react'
import clsx from 'clsx'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '../../auth/AuthContext'
import { useNotifications } from '../../notifications/hooks/useNotifications'
import type { Notification } from '../../notifications/types'
import { useDateFormat } from '../../format/DateFormatProvider'

export default function NotificationBell() {
  const { token } = useAuth()
  const { t } = useTranslation(['layout', 'common'])
  const { formatDateTime } = useDateFormat()
  const { notifications, unreadCount, loading, error, refresh, markRead, markAllRead } = useNotifications(token ?? '')
  const [open, setOpen] = useState(false)
  const navigate = useNavigate()

  function handleClick(n: Notification) {
    if (!n.read) markRead(n.id)
    if (n.link) {
      setOpen(false)
      navigate(n.link)
    }
  }

  function toggleOpen() {
    const next = !open
    setOpen(next)
    if (next) refresh()
  }

  return (
    <div className="relative">
      <button
        onClick={toggleOpen}
        aria-label={unreadCount > 0 ? t('ui.notificationsUnread', { count: unreadCount }) : t('ui.notifications')}
        aria-haspopup="true"
        aria-expanded={open}
        className="relative w-8 h-8 flex items-center justify-center rounded-lg hover:bg-hover transition-colors"
      >
        <Bell size={16} className="text-ink-3" />
        {unreadCount > 0 && (
          <span className="absolute top-1 right-1 min-w-[14px] h-[14px] px-0.5 flex items-center justify-center bg-red-500 text-white text-2xs font-medium rounded-full">
            {unreadCount > 9 ? '9+' : unreadCount}
          </span>
        )}
      </button>

      {open && (
        <>
          <div className="fixed inset-0 z-40" onClick={() => setOpen(false)} />
          <div
            onKeyDown={e => { if (e.key === 'Escape') setOpen(false) }}
            className="absolute right-0 top-10 z-50 w-80 max-h-96 flex flex-col bg-raised rounded-xl shadow-2xl border border-line overflow-hidden"
          >
            <div className="flex items-center justify-between px-3 py-2.5 border-b border-line shrink-0">
              <p className="text-xs font-semibold text-ink">{t('ui.notifications')}</p>
              {unreadCount > 0 && (
                <button
                  onClick={() => markAllRead()}
                  className="text-2xs font-medium text-accent hover:text-accent/80 transition-colors"
                >
                  {t('ui.markAllRead')}
                </button>
              )}
            </div>

            <div className="flex-1 overflow-y-auto">
              {loading && <p className="p-4 text-xs text-center text-ink-4">{t('common:status.loading')}</p>}
              {error && <p className="p-4 text-xs text-center text-danger">{error}</p>}
              {!loading && !error && notifications.length === 0 && (
                <p className="p-4 text-xs text-center text-ink-4">{t('ui.noNotifications')}</p>
              )}
              {notifications.map(n => (
                <button
                  key={n.id}
                  onClick={() => handleClick(n)}
                  className={clsx(
                    'w-full text-left px-3 py-2.5 border-b border-line/60 hover:bg-sunken transition-colors',
                    !n.read && 'bg-accent/10',
                  )}
                >
                  <div className="flex items-start gap-2">
                    {!n.read && <span className="mt-1 w-1.5 h-1.5 rounded-full bg-accent-solid shrink-0" />}
                    <div className={clsx('min-w-0 flex-1', n.read && 'pl-3.5')}>
                      <p className="text-xs font-medium text-ink truncate">{n.title}</p>
                      {n.message && <p className="text-2xs text-ink-4 mt-0.5 line-clamp-2">{n.message}</p>}
                      <p className="text-2xs text-ink-4 mt-1">{formatDateTime(n.created_at)}</p>
                    </div>
                  </div>
                </button>
              ))}
            </div>
          </div>
        </>
      )}
    </div>
  )
}
