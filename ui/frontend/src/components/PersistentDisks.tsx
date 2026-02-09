import { useEffect, useState } from 'react'
import { HardDrive, WifiOff } from 'lucide-react'
import type { K8sPVC } from '../types'
import { cn } from '@/lib/utils'
import { usePanelLoading } from '../hooks/useLoadingTracker'

function parseStorage(size: string): number {
  const match = size.match(/^(\d+)(Gi|Mi|Ti)?$/)
  if (!match) return 0
  const val = parseInt(match[1])
  const unit = match[2]
  if (unit === 'Ti') return val * 1024
  if (unit === 'Gi') return val
  if (unit === 'Mi') return val / 1024
  return val
}

function formatStorage(gb: number): string {
  if (gb >= 1024) return `${(gb / 1024).toFixed(1)} TB`
  if (gb >= 1) return `${gb} GB`
  return `${Math.round(gb * 1024)} MB`
}

export function PersistentDisks({ isStale }: { isStale?: boolean }) {
  const [pvcs, setPvcs] = useState<K8sPVC[]>([])
  const [loading, setLoading] = useState(true)
  const { setLoading: trackStart, setLoaded } = usePanelLoading('pvcs', 'Fetching persistent disks...')

  useEffect(() => {
    trackStart()
    const fetchData = () => {
      fetch('/api/pvcs?ns=application')
        .then(res => res.json())
        .then(data => setPvcs(data.items || []))
        .catch(console.error)
        .finally(() => { setLoading(false); setLoaded() })
    }

    fetchData()
    const interval = setInterval(fetchData, 15000)
    return () => clearInterval(interval)
  }, [])

  if (loading) return (
    <div className="border rounded-xl bg-card shadow-sm overflow-hidden">
      <div className="p-4 border-b">
        <h2 className="font-semibold flex items-center gap-2">
          <HardDrive className="w-4 h-4" /> Persistent Disks
        </h2>
      </div>
      <div className="p-4 space-y-3 animate-pulse">
        {[1, 2].map(i => (
          <div key={i} className="space-y-1.5">
            <div className="flex justify-between"><div className="h-4 bg-muted rounded w-1/2" /><div className="h-4 bg-muted rounded w-16" /></div>
            <div className="w-full h-2 bg-muted rounded-full" />
          </div>
        ))}
      </div>
    </div>
  )
  if (pvcs.length === 0) return (
    <div className="p-6 border rounded-xl bg-card shadow-sm text-center text-muted-foreground text-sm">
      No persistent disks found
    </div>
  )

  const maxGb = Math.max(...pvcs.map(p => parseStorage(p.spec.resources.requests.storage)), 1)

  return (
    <div className="border rounded-xl bg-card shadow-sm overflow-hidden">
      <div className="p-4 border-b">
        <h2 className="font-semibold flex items-center gap-2">
          <HardDrive className="w-4 h-4" /> Persistent Disks
        </h2>
      </div>
      <div className="p-4 space-y-3">
        {pvcs.map(pvc => {
          const gb = parseStorage(pvc.spec.resources.requests.storage)
          const pct = Math.round((gb / maxGb) * 100)
          return (
            <div key={pvc.metadata.name} className="space-y-1.5">
              <div className="flex items-center justify-between text-sm">
                <span className="font-medium truncate">{pvc.metadata.name}</span>
                <div className="flex items-center gap-2">
                  <span className={cn(
                    "px-2 py-0.5 rounded text-xs font-bold",
                    pvc.status.phase === 'Bound' ? "bg-green-500/10 text-green-500" : "bg-yellow-500/10 text-yellow-500"
                  )}>{pvc.status.phase}</span>
                  <span className="font-mono text-xs text-muted-foreground">{formatStorage(gb)}</span>
                </div>
              </div>
              <div className="w-full h-2 bg-muted rounded-full overflow-hidden">
                <div
                  className="h-full bg-primary/60 rounded-full transition-all"
                  style={{ width: `${pct}%` }}
                />
              </div>
            </div>
          )
        })}
      </div>
      {isStale && (
        <div className="px-4 py-2 bg-yellow-500/10 border-t border-yellow-500/20 text-yellow-600 text-xs flex items-center gap-2">
          <WifiOff className="w-3 h-3" /> Data may be stale. Tunnel disconnected.
        </div>
      )}
    </div>
  )
}
