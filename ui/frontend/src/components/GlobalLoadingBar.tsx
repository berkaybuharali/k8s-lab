import { Loader2 } from 'lucide-react'
import { useLoadingTracker } from '../hooks/useLoadingTracker'

export function GlobalLoadingBar() {
  const { isLoading, pendingLabels } = useLoadingTracker()

  if (!isLoading) return null

  const currentLabel = pendingLabels[0] || 'Loading...'

  return (
    <div className="fixed bottom-6 right-6 z-40 flex items-center gap-3 px-5 py-3 rounded-xl border bg-card shadow-lg">
      <Loader2 className="w-5 h-5 animate-spin text-primary flex-shrink-0" />
      <span className="text-sm font-medium text-foreground">
        {currentLabel}
      </span>
    </div>
  )
}
