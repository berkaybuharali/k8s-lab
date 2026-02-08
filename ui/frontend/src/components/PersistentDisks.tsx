import { useEffect, useState } from 'react'
import { HardDrive } from 'lucide-react'
import type { K8sPVC } from '../types'
import { cn } from '@/lib/utils'

export function PersistentDisks() {
  const [pvcs, setPvcs] = useState<K8sPVC[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const fetchData = () => {
      fetch('/api/pvcs?ns=application')
        .then(res => res.json())
        .then(data => setPvcs(data.items || []))
        .catch(console.error)
        .finally(() => setLoading(false))
    }

    fetchData()
    const interval = setInterval(fetchData, 15000)
    return () => clearInterval(interval)
  }, [])

  if (loading) return null
  if (pvcs.length === 0) return (
    <div className="p-6 border rounded-lg bg-card shadow-sm text-center text-muted-foreground text-sm">
      No persistent disks found
    </div>
  )

  return (
    <div className="border rounded-lg bg-card shadow-sm overflow-hidden">
      <div className="p-4 border-b">
        <h2 className="font-semibold flex items-center gap-2">
          <HardDrive className="w-4 h-4" /> Persistent Disks
        </h2>
      </div>
      <table className="w-full text-sm text-left">
        <thead className="bg-muted/50 text-xs text-muted-foreground">
          <tr>
            <th className="p-3">Name</th>
            <th className="p-3">Status</th>
            <th className="p-3">Size</th>
          </tr>
        </thead>
        <tbody className="divide-y">
          {pvcs.map(pvc => (
            <tr key={pvc.metadata.name}>
              <td className="p-3 font-medium">{pvc.metadata.name}</td>
              <td className="p-3">
                <span className={cn(
                  "px-2 py-0.5 rounded text-xs font-bold",
                  pvc.status.phase === 'Bound' ? "bg-green-500/10 text-green-500" : "bg-yellow-500/10 text-yellow-500"
                )}>{pvc.status.phase}</span>
              </td>
              <td className="p-3 font-mono text-xs">{pvc.spec.resources.requests.storage}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
