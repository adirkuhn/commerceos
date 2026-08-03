import { useState } from 'react'
import { Bell } from 'lucide-react'
import clsx from 'clsx'
import { useAuth } from '../../auth/AuthContext'
import { useNotifications } from '../../notifications/hooks/useNotifications'

function formatTime(iso: string): string {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return ''
  return d.toLocaleString()
}

export default function NotificationBell() {
  const { token } = useAuth()
  const { notifications, unreadCount, loading, error, refresh, markRead, markAllRead } = useNotifications(token ?? '')
  const [open, setOpen] = useState(false)

  function toggleOpen() {
    const next = !open
    setOpen(next)
    if (next) refresh()
  }

  return (
    <div className="relative">
      <button
        onClick={toggleOpen}
        aria-label="Notifications"
        className="relative w-8 h-8 flex items-center justify-center rounded-lg dark:hover:bg-[#1F2937] hover:bg-gray-100 transition-colors"
      >
        <Bell size={16} className="dark:text-gray-400 text-gray-500" />
        {unreadCount > 0 && (
          <span className="absolute top-1 right-1 min-w-[14px] h-[14px] px-0.5 flex items-center justify-center bg-red-500 text-white text-[9px] font-medium rounded-full">
            {unreadCount > 9 ? '9+' : unreadCount}
          </span>
        )}
      </button>

      {open && (
        <>
          <div className="fixed inset-0 z-40" onClick={() => setOpen(false)} />
          <div className="absolute right-0 top-10 z-50 w-80 max-h-96 flex flex-col dark:bg-[#1F2937] bg-white rounded-xl shadow-2xl border dark:border-[#2D3748] border-gray-200 overflow-hidden">
            <div className="flex items-center justify-between px-3 py-2.5 border-b dark:border-[#2D3748] border-gray-200 shrink-0">
              <p className="text-xs font-semibold dark:text-gray-200 text-gray-800">Notifications</p>
              {unreadCount > 0 && (
                <button
                  onClick={() => markAllRead()}
                  className="text-[10px] font-medium text-blue-400 hover:text-blue-300 transition-colors"
                >
                  Mark all read
                </button>
              )}
            </div>

            <div className="flex-1 overflow-y-auto">
              {loading && <p className="p-4 text-xs text-center dark:text-gray-500 text-gray-400">Loading…</p>}
              {error && <p className="p-4 text-xs text-center text-red-400">{error}</p>}
              {!loading && !error && notifications.length === 0 && (
                <p className="p-4 text-xs text-center dark:text-gray-500 text-gray-400">No notifications yet.</p>
              )}
              {notifications.map(n => (
                <button
                  key={n.id}
                  onClick={() => !n.read && markRead(n.id)}
                  className={clsx(
                    'w-full text-left px-3 py-2.5 border-b dark:border-[#2D3748]/60 border-gray-100 hover:dark:bg-[#111827] hover:bg-gray-50 transition-colors',
                    !n.read && 'dark:bg-blue-500/5 bg-blue-50/50',
                  )}
                >
                  <div className="flex items-start gap-2">
                    {!n.read && <span className="mt-1 w-1.5 h-1.5 rounded-full bg-blue-500 shrink-0" />}
                    <div className={clsx('min-w-0 flex-1', n.read && 'pl-3.5')}>
                      <p className="text-xs font-medium dark:text-gray-200 text-gray-800 truncate">{n.title}</p>
                      {n.message && <p className="text-[10px] dark:text-gray-500 text-gray-400 mt-0.5 line-clamp-2">{n.message}</p>}
                      <p className="text-[9px] dark:text-gray-600 text-gray-400 mt-1">{formatTime(n.created_at)}</p>
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
