import { useEffect, useState } from 'react'
import { Archive, Trash2, RotateCcw, WifiOff } from 'lucide-react'
import type { VeleroBackup } from '../types'
import { cn } from '@/lib/utils'

export function BackupList({ isStale }: { isStale?: boolean }) {
  const [backups, setBackups] = useState<VeleroBackup[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const fetchData = () => {
      fetch('/api/backups')
        .then(res => res.json())
        .then(data => setBackups(data.items || []))
        .catch(console.error)
        .finally(() => setLoading(false))
    }

    fetchData()
    const interval = setInterval(fetchData, 15000)
    return () => clearInterval(interval)
  }, [])

  if (loading) return null

  return (
    <div className="border rounded-lg bg-card shadow-sm overflow-hidden">
      <div className="p-4 border-b flex items-center justify-between">
        <h2 className="font-semibold flex items-center gap-2">
          <Archive className="w-4 h-4" /> Backups
        </h2>
        <div className="text-xs text-muted-foreground">{backups.length} items</div>
      </div>
      
      {backups.length === 0 ? (
        <div className="p-8 text-center text-muted-foreground text-sm">
          No backups available
        </div>
      ) : (
        <table className="w-full text-sm text-left">
          <thead className="bg-muted/50 text-xs text-muted-foreground">
            <tr>
              <th className="p-3">Name</th>
              <th className="p-3">Status</th>
              <th className="p-3">Expires</th>
              <th className="p-3 text-right">Actions</th>
            </tr>
          </thead>
          <tbody className="divide-y">
            {backups.map(backup => (
              <tr key={backup.metadata.name}>
                <td className="p-3 font-medium">{backup.metadata.name}</td>
                <td className="p-3">
                  <span className={cn(
                    "px-2 py-0.5 rounded text-xs font-bold",
                    backup.status.phase === 'Completed' ? "bg-green-500/10 text-green-500" : "bg-yellow-500/10 text-yellow-500"
                  )}>{backup.status.phase}</span>
                </td>
                <td className="p-3 text-xs text-muted-foreground">
                  {new Date(backup.status.expiration).toLocaleDateString()}
                </td>
                <td className="p-3 text-right">
                  <button className="p-1 hover:bg-muted rounded text-blue-500 mr-1" title="Restore">
                    <RotateCcw className="w-4 h-4" />
                  </button>
                  <button className="p-1 hover:bg-muted rounded text-red-500" title="Delete">
                    <Trash2 className="w-4 h-4" />
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
      {isStale && (
        <div className="px-4 py-2 bg-yellow-500/10 border-t border-yellow-500/20 text-yellow-600 text-xs flex items-center gap-2">
          <WifiOff className="w-3 h-3" /> Data may be stale. Tunnel disconnected.
        </div>
      )}
    </div>
  )
}
