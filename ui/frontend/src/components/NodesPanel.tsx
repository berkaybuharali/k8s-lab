import { useEffect, useState } from 'react'
import { Activity, WifiOff } from 'lucide-react'
import type { K8sNode } from '../types'
import { cn } from '@/lib/utils'
import { usePanelLoading } from '../hooks/useLoadingTracker'
import kubernetesLogo from '@/assets/kubernetes_logo.svg'
import talosLogo from '@/assets/talos_logo.svg'
import gceLogo from '@/assets/gce_logo.svg'

export function NodesPanel({ isStale, provider }: { isStale?: boolean; provider?: string }) {
  const [nodes, setNodes] = useState<K8sNode[]>([])
  const [loading, setLoading] = useState(true)
  const { setLoading: trackStart, setLoaded } = usePanelLoading('nodes')

  useEffect(() => {
    trackStart()
    const fetchData = () => {
      fetch('/api/nodes')
        .then(res => res.json())
        .then(data => setNodes(data.items || []))
        .catch(console.error)
        .finally(() => { setLoading(false); setLoaded() })
    }

    fetchData()
    const interval = setInterval(fetchData, 15000)
    return () => clearInterval(interval)
  }, [])

  if (loading) return (
    <div className="space-y-4">
      <h2 className="font-semibold flex items-center gap-2">
        <img src={kubernetesLogo} alt="Kubernetes" className="w-4 h-4" /> Nodes
      </h2>
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {[1, 2, 3].map(i => (
          <div key={i} className="border rounded-xl p-4 bg-card shadow-sm animate-pulse">
            <div className="h-4 bg-muted rounded w-3/4 mb-3" />
            <div className="space-y-2">
              <div className="h-3 bg-muted rounded w-1/2" />
              <div className="h-3 bg-muted rounded w-2/3" />
              <div className="h-3 bg-muted rounded w-1/3" />
            </div>
          </div>
        ))}
      </div>
    </div>
  )
  if (nodes.length === 0) return null

  return (
    <div className="space-y-4">
      <h2 className="font-semibold flex items-center gap-2">
        <img src={kubernetesLogo} alt="Kubernetes" className="w-4 h-4" /> Nodes ({nodes.length})
      </h2>
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {nodes.map((node) => {
          const ready = node.status.conditions.find(c => c.type === 'Ready')?.status === 'True'
          const ip = node.status.addresses.find(a => a.type === 'InternalIP')?.address
          const role = Object.keys(node.metadata.labels).find(l => l.includes('node-role.kubernetes.io'))?.split('/')[1] || 'worker'

          return (
            <div key={node.metadata.name} className="border rounded-xl p-4 bg-card shadow-sm">
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
                  <img src={talosLogo} alt="Talos" className="w-3 h-3" /> {node.status.nodeInfo.osImage}
                </div>
                <div className="flex items-center gap-2 font-mono">
                  {provider === 'gcp' && <img src={gceLogo} alt="GCE" className="w-3 h-3" />}
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
