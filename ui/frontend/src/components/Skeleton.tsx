import { cn } from '@/lib/utils'

export function Skeleton({ className }: { className?: string }) {
  return <div className={cn("animate-pulse rounded bg-muted", className)} />
}

export function PanelSkeleton({ title, rows = 3 }: { title: string; rows?: number }) {
  return (
    <div className="border rounded-xl bg-card shadow-sm overflow-hidden">
      <div className="p-4 border-b">
        <h2 className="font-semibold text-sm flex items-center gap-2">
          <Skeleton className="w-4 h-4 rounded" />
          {title}
        </h2>
      </div>
      <div className="p-4 space-y-3">
        {Array.from({ length: rows }).map((_, i) => (
          <div key={i} className="flex items-center gap-3">
            <Skeleton className="h-4 flex-1" />
            <Skeleton className="h-4 w-16" />
          </div>
        ))}
      </div>
    </div>
  )
}
