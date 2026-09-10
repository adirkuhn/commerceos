import type { ReactNode } from 'react'
import { AlertTriangle, Inbox, WifiOff, Lock, SearchX } from 'lucide-react'
import { Button } from './Button'
import { Skeleton } from './Skeleton'

interface StateProps {
  title: string
  description?: string
  action?: ReactNode
}

export function LoadingState({ label = 'Loading…' }: { label?: string }) {
  return (
    <div role="status" className="flex flex-col items-center justify-center gap-3 py-16 text-muted-foreground">
      <div className="h-8 w-8 animate-spin rounded-full border-2 border-current border-t-transparent" />
      <p className="text-sm">{label}</p>
    </div>
  )
}

export function EmptyState({ title, description, action, icon }: StateProps & { icon?: ReactNode }) {
  return (
    <div className="flex flex-col items-center justify-center gap-3 rounded-[var(--radius)] border border-dashed border-border py-16 text-center">
      <div className="text-muted-foreground">{icon ?? <Inbox className="h-8 w-8" />}</div>
      <p className="text-sm font-medium">{title}</p>
      {description && <p className="max-w-sm text-sm text-muted-foreground">{description}</p>}
      {action}
    </div>
  )
}

export function ErrorState({ title = 'Something went wrong', description, onRetry }: StateProps & { onRetry?: () => void }) {
  return (
    <div role="alert" className="flex flex-col items-center justify-center gap-3 rounded-[var(--radius)] border border-destructive/30 bg-destructive/5 py-16 text-center">
      <AlertTriangle className="h-8 w-8 text-destructive" />
      <p className="text-sm font-medium">{title}</p>
      {description && <p className="max-w-sm text-sm text-muted-foreground">{description}</p>}
      {onRetry && <Button variant="outline" size="sm" onClick={onRetry}>Try again</Button>}
    </div>
  )
}

export function OfflineState({ onRetry }: { onRetry?: () => void }) {
  return <ErrorState title="You're offline" description="Check your connection and try again." onRetry={onRetry} />
}

export function PermissionState({ description = 'You do not have access to this page.' }: { description?: string }) {
  return <EmptyState icon={<Lock className="h-8 w-8" />} title="Access denied" description={description} />
}

export function NoResultsState({ query }: { query?: string }) {
  return (
    <EmptyState
      icon={<SearchX className="h-8 w-8" />}
      title={query ? `No results for "${query}"` : 'No results'}
      description="Try a different search term or browse categories instead."
    />
  )
}

export function NotFound({ title = 'Page not found', description = "The page you're looking for doesn't exist." }: { title?: string; description?: string }) {
  return (
    <div className="flex min-h-[50vh] flex-col items-center justify-center gap-3 text-center">
      <WifiOff className="h-10 w-10 text-muted-foreground" />
      <h1 className="text-2xl font-semibold">{title}</h1>
      <p className="max-w-sm text-muted-foreground">{description}</p>
      <Button asChild><a href="/">Back to home</a></Button>
    </div>
  )
}

export function ProductSkeleton() {
  return (
    <div className="space-y-3">
      <Skeleton className="aspect-square w-full" />
      <Skeleton className="h-4 w-3/4" />
      <Skeleton className="h-4 w-1/3" />
    </div>
  )
}

export function ProductGridSkeleton({ count = 8 }: { count?: number }) {
  return (
    <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4">
      {Array.from({ length: count }, (_, i) => <ProductSkeleton key={i} />)}
    </div>
  )
}
