import { X } from 'lucide-react'
import { useToastStore } from './toast'
import { cn } from '@/lib/utils'

export function Toaster() {
  const toasts = useToastStore(s => s.toasts)
  const dismiss = useToastStore(s => s.dismiss)

  if (toasts.length === 0) return null

  return (
    <div className="fixed bottom-4 right-4 z-[100] flex w-full max-w-sm flex-col gap-2" role="region" aria-label="Notifications">
      {toasts.map(t => (
        <div
          key={t.id}
          role="status"
          className={cn(
            'animate-fade-in flex items-start justify-between gap-3 rounded-[var(--radius)] border border-border bg-card p-4 shadow-lg',
            t.variant === 'destructive' && 'border-destructive bg-destructive text-destructive-foreground',
            t.variant === 'success' && 'border-emerald-600 bg-emerald-600 text-white',
          )}
        >
          <div>
            <p className="text-sm font-medium">{t.title}</p>
            {t.description && <p className="mt-0.5 text-xs opacity-90">{t.description}</p>}
          </div>
          <button onClick={() => dismiss(t.id)} aria-label="Dismiss notification" className="shrink-0 opacity-70 hover:opacity-100">
            <X className="h-4 w-4" />
          </button>
        </div>
      ))}
    </div>
  )
}
