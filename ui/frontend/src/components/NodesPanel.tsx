import { useEffect, useState } from 'react'
import { Server, Cpu, Activity, WifiOff } from 'lucide-react'
import type { K8sNode } from '../types'
import { cn } from '@/lib/utils'

export function NodesPanel({ isStale }: { isStale?: boolean }) {
  const [nodes, setNodes] = useState<K8sNode[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const fetchData = () => {
      fetch('/api/nodes')
        .then(res => res.json())
        .then(data => setNodes(data.items || []))
        .catch(console.error)
        .finally(() => setLoading(false))
    }

    fetchData()
    const interval = setInterval(fetchData, 15000)
    return () => clearInterval(interval)
  }, [])

  if (loading) return <div className="p-4 text-sm text-muted-foreground">Loading nodes...</div>
  if (nodes.length === 0) return null

  return (
    <div className="space-y-4">
      <h2 className="font-semibold flex items-center gap-2">
        <Server className="w-4 h-4" /> Nodes ({nodes.length})
      </h2>
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {nodes.map((node) => {
          const ready = node.status.conditions.find(c => c.type === 'Ready')?.status === 'True'
          const ip = node.status.addresses.find(a => a.type === 'InternalIP')?.address
          const role = Object.keys(node.metadata.labels).find(l => l.includes('node-role.kubernetes.io'))?.split('/')[1] || 'worker'

          return (
            <div key={node.metadata.name} className="border rounded-lg p-4 bg-card shadow-sm">
              <div className="flex justify-between items-start mb-2">
                <div className="font-medium truncate" title={node.metadata.name}>{node.metadata.name}</div>
                <div className={cn("px-2 py-0.5 rounded text-xs font-bold", ready ? "bg-green-500/10 text-green-500" : "bg-red-500/10 text-red-500")}>
                  {ready ? 'Ready' : 'Not Ready'}
                </div>
              </div>
              <div className="space-y-1 text-xs text-muted-foreground">
                <div className="flex items-center gap-2">
                  <Activity className="w-3 h-3" /> {role}
                </div>
                <div className="flex items-center gap-2">
                  <Cpu className="w-3 h-3" /> {node.status.nodeInfo.osImage}
                </div>
                <div className="flex items-center gap-2 font-mono">
                  {ip}
                </div>
              </div>
            </div>
          )
        })}
      </div>
      {isStale && (
        <div className="px-4 py-2 bg-yellow-500/10 border border-yellow-500/20 rounded-md text-yellow-600 text-xs flex items-center gap-2 mt-2">
          <WifiOff className="w-3 h-3" /> Data may be stale. Tunnel disconnected.
        </div>
      )}
    </div>
  )
}
