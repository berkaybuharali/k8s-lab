import { useEffect, useState } from 'react'
import { WifiOff } from 'lucide-react'
import { cn } from '@/lib/utils'
import { usePanelLoading } from '../hooks/useLoadingTracker'
import gcpDiskLogo from '@/assets/gcp_persistent_disk.svg'

interface Snapshot {
  name: string
  status: string
  diskSizeGb: string
  storageBytes: string
  sourceDisk: string
  creationTimestamp: string
}

interface DiskSnapshotsProps {
  isStale?: boolean
  provider?: string
  refreshTrigger?: number
}

function formatBytes(bytes: string): string {
  const b = parseInt(bytes)
  if (isNaN(b) || b === 0) return '0 B'
  if (b >= 1073741824) return `${(b / 1073741824).toFixed(1)} GB`
  if (b >= 1048576) return `${(b / 1048576).toFixed(1)} MB`
  if (b >= 1024) return `${(b / 1024).toFixed(1)} KB`
  return `${b} B`
}

export function DiskSnapshots({ isStale, provider, refreshTrigger }: DiskSnapshotsProps) {
  const [snapshots, setSnapshots] = useState<Snapshot[]>([])
  const [loading, setLoading] = useState(true)
  const { setLoading: trackStart, setLoaded } = usePanelLoading('snapshots', 'Fetching disk snapshots...')

  useEffect(() => {
    if (provider !== 'gcp') {
      setLoading(false)
      setLoaded()
      return
    }

    trackStart()
    const fetchData = () => {
      fetch('/api/snapshots')
        .then(res => res.json())
        .then(data => setSnapshots(Array.isArray(data) ? data : []))
        .catch(console.error)
        .finally(() => { setLoading(false); setLoaded() })
    }

    fetchData()
    const interval = setInterval(fetchData, 30000)
    return () => clearInterval(interval)
  }, [provider, refreshTrigger])

  if (provider !== 'gcp') return null
  if (loading) return (
    <div className="border rounded-xl bg-card shadow-sm overflow-hidden">
      <div className="p-4 border-b">
        <h2 className="font-semibold flex items-center gap-2">
          <img src={gcpDiskLogo} alt="Disk" className="w-4 h-4" /> Disk Snapshots
        </h2>
      </div>
      <div className="p-4 space-y-3 animate-pulse">
        {[1, 2].map(i => (
          <div key={i} className="flex items-center gap-3">
            <div className="h-4 bg-muted rounded flex-1" />
            <div className="h-4 bg-muted rounded w-16" />
            <div className="h-4 bg-muted rounded w-20" />
          </div>
        ))}
      </div>
    </div>
  )
  if (snapshots.length === 0) return (
    <div className="p-6 border rounded-xl bg-card shadow-sm text-center text-muted-foreground text-sm">
      <div className="flex items-center justify-center gap-2 mb-1">
        <img src={gcpDiskLogo} alt="Disk" className="w-4 h-4" /> Disk Snapshots
      </div>
      No snapshots found
    </div>
  )

  return (
    <div className="border rounded-xl bg-card shadow-sm overflow-hidden">
      <div className="p-4 border-b flex items-center justify-between">
        <h2 className="font-semibold flex items-center gap-2">
          <img src={gcpDiskLogo} alt="Disk" className="w-4 h-4" /> Disk Snapshots
        </h2>
        <div className="text-xs text-muted-foreground">{snapshots.length} items</div>
      </div>

      <table className="w-full text-sm text-left">
        <thead className="bg-muted/50 text-xs text-muted-foreground">
          <tr>
            <th className="p-3">Name</th>
            <th className="p-3">Status</th>
            <th className="p-3">Size</th>
            <th className="p-3">Created</th>
          </tr>
        </thead>
        <tbody className="divide-y">
          {snapshots.map(snap => (
            <tr key={snap.name}>
              <td className="p-3 font-medium truncate max-w-[200px]" title={snap.name}>{snap.name}</td>
              <td className="p-3">
                <span className={cn(
                  "px-2 py-0.5 rounded text-xs font-bold",
                  snap.status === 'READY' ? "bg-green-500/10 text-green-500" : "bg-yellow-500/10 text-yellow-500"
                )}>{snap.status}</span>
              </td>
              <td className="p-3 font-mono text-xs">
                {formatBytes(snap.storageBytes || '0')} / {snap.diskSizeGb || '?'} GB
              </td>
              <td className="p-3 text-xs text-muted-foreground">
                {new Date(snap.creationTimestamp).toLocaleDateString()}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
      {isStale && (
        <div className="px-4 py-2 bg-yellow-500/10 border-t border-yellow-500/20 text-yellow-600 text-xs flex items-center gap-2">
          <WifiOff className="w-3 h-3" /> Data may be stale.
        </div>
      )}
    </div>
  )
}
