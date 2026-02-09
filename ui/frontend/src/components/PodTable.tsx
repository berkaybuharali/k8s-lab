import { useEffect, useState } from 'react'
import { RotateCw, WifiOff } from 'lucide-react'
import kubernetesLogo from '@/assets/kubernetes_logo.svg'
import nginxLogo from '@/assets/nginx_logo.svg'
import redisLogo from '@/assets/redis_logo.svg'
import { usePanelLoading } from '../hooks/useLoadingTracker'

function getPodLogo(name: string): string | null {
  if (name.includes('nginx')) return nginxLogo
  if (name.includes('redis')) return redisLogo
  return null
}
import type { K8sPod } from '../types'
import { cn } from '@/lib/utils'

interface PodTableProps {
  isStale?: boolean
  onPodClick: (name: string, ns: string) => void
}

export function PodTable({ isStale, onPodClick }: PodTableProps) {
  const [pods, setPods] = useState<K8sPod[]>([])
  const [namespaces, setNamespaces] = useState<string[]>([])
  const [ns, setNs] = useState('application')
  const [loading, setLoading] = useState(true)
  const { setLoading: trackStart, setLoaded } = usePanelLoading('pods')

  useEffect(() => {
    trackStart()
    // Fetch namespaces
    fetch('/api/namespaces')
      .then(res => res.json())
      .then(data => {
        const list = data.items?.map((n: any) => n.metadata.name) || []
        setNamespaces(list)
      })
      .catch(console.error)
  }, [])

  useEffect(() => {
    setLoading(true)
    const fetchData = () => {
      fetch(`/api/pods?ns=${ns}`)
        .then(res => res.json())
        .then(data => setPods(data.items || []))
        .catch(console.error)
        .finally(() => { setLoading(false); setLoaded() })
    }

    fetchData()
    const interval = setInterval(fetchData, 15000)
    return () => clearInterval(interval)
  }, [ns])

  return (
    <div className="border rounded-xl bg-card shadow-sm flex flex-col h-[400px]">
      <div className="p-4 border-b flex items-center justify-between">
        <h2 className="font-semibold flex items-center gap-2">
          <img src={kubernetesLogo} alt="K8s" className="w-4 h-4" /> Pods
        </h2>
        <select 
          value={ns} 
          onChange={(e) => setNs(e.target.value)}
          disabled={isStale}
          className={cn(
            "text-xs border rounded px-2 py-1 bg-background",
            isStale && "opacity-50 cursor-not-allowed"
          )}
        >
          {namespaces.map(n => <option key={n} value={n}>{n}</option>)}
        </select>
      </div>
      
      <div className="flex-1 overflow-auto">
        <table className="w-full text-sm text-left">
          <thead className="text-xs text-muted-foreground bg-muted/50 sticky top-0">
            <tr>
              <th className="p-3 font-medium">Name</th>
              <th className="p-3 font-medium">Status</th>
              <th className="p-3 font-medium">Restarts</th>
              <th className="p-3 font-medium">Age</th>
            </tr>
          </thead>
          <tbody className="divide-y">
            {loading && Array.from({ length: 4 }).map((_, i) => (
              <tr key={i} className="animate-pulse">
                <td className="p-3"><div className="h-4 bg-muted rounded w-3/4" /></td>
                <td className="p-3"><div className="h-4 bg-muted rounded w-16" /></td>
                <td className="p-3"><div className="h-4 bg-muted rounded w-8" /></td>
                <td className="p-3"><div className="h-4 bg-muted rounded w-10" /></td>
              </tr>
            ))}
            {!loading && pods.length === 0 && <tr><td colSpan={4} className="p-4 text-center text-muted-foreground">No pods found</td></tr>}
            
            {!loading && pods.map(pod => {
              const status = pod.status.phase
              const restarts = pod.status.containerStatuses?.reduce((acc, c) => acc + c.restartCount, 0) || 0
              const age = new Date().getTime() - new Date(pod.metadata.creationTimestamp).getTime()
              const ageStr = age > 3600000 ? `${Math.floor(age/3600000)}h` : `${Math.floor(age/60000)}m`

              return (
                <tr 
                  key={pod.metadata.name} 
                  onClick={() => onPodClick(pod.metadata.name, ns)}
                  className="hover:bg-muted/50 transition-colors cursor-pointer"
                >
                  <td className="p-3 font-medium flex items-center gap-2">
                    {getPodLogo(pod.metadata.name) && <img src={getPodLogo(pod.metadata.name)!} alt="" className="w-4 h-4 flex-shrink-0" />}
                    {pod.metadata.name}
                  </td>
                  <td className="p-3">
                    <span className={cn(
                      "px-2 py-0.5 rounded text-xs font-bold",
                      status === 'Running' ? "bg-green-500/10 text-green-500" : 
                      status === 'Pending' ? "bg-yellow-500/10 text-yellow-500" : "bg-red-500/10 text-red-500"
                    )}>{status}</span>
                  </td>
                  <td className="p-3 flex items-center gap-1">
                    <RotateCw className="w-3 h-3 text-muted-foreground" /> {restarts}
                  </td>
                  <td className="p-3 text-muted-foreground">
                    {ageStr}
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>
      {isStale && (
        <div className="px-4 py-2 bg-yellow-500/10 border-t border-yellow-500/20 text-yellow-600 text-xs flex items-center gap-2">
          <WifiOff className="w-3 h-3" /> Data may be stale. Tunnel disconnected.
        </div>
      )}
    </div>
  )
}