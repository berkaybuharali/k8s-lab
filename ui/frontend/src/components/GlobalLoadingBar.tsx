import { Loader2 } from 'lucide-react'
import { useLoadingTracker } from '../hooks/useLoadingTracker'

export function GlobalLoadingBar() {
  const { isLoading, pendingCount, totalCount } = useLoadingTracker()

  if (!isLoading) return null

  const loaded = totalCount - pendingCount

  return (
    <div className="shrink-0 border-b bg-primary/5">
      <div className="flex items-center gap-3 px-4 py-2">
        <Loader2 className="w-4 h-4 animate-spin text-primary" />
        <div className="flex items-center gap-3 flex-1">
          <span className="text-xs font-medium text-muted-foreground">
            Fetching cluster data... ({loaded}/{totalCount})
          </span>
          <div className="w-32 h-1.5 bg-muted rounded-full overflow-hidden">
            <div
              className="h-full bg-primary rounded-full transition-all duration-700 ease-out"
              style={{ width: `${totalCount > 0 ? Math.max(Math.round((loaded / totalCount) * 100), 8) : 8}%` }}
            />
          </div>
        </div>
      </div>
    </div>
  )
}
