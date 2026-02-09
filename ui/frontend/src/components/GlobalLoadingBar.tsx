import { Loader2 } from 'lucide-react'
import { useLoadingTracker } from '../hooks/useLoadingTracker'

export function GlobalLoadingBar() {
  const { isLoading, pendingLabels } = useLoadingTracker()

  if (!isLoading) return null

  // Show the first pending label as the current activity
  const currentLabel = pendingLabels[0] || 'Loading...'

  return (
    <div className="shrink-0 border-b bg-primary/5">
      <div className="flex items-center gap-3 px-4 py-2">
        <Loader2 className="w-4 h-4 animate-spin text-primary flex-shrink-0" />
        <span className="text-xs font-medium text-muted-foreground">
          {currentLabel}
        </span>
      </div>
    </div>
  )
}
