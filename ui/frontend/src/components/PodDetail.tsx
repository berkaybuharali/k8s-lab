import { useEffect, useState } from 'react'
import { ArrowLeft, Terminal, Info, RotateCw, Cpu, Clock, Box, Network } from 'lucide-react'
import type { K8sPod } from '../types'
import { cn } from '@/lib/utils'

interface PodDetailProps {
  podName: string
  namespace: string
  onBack: () => void
}

export function PodDetail({ podName, namespace, onBack }: PodDetailProps) {
  const [pod, setPod] = useState<K8sPod | null>(null)
  const [logs, setLogs] = useState<string>('')
  const [tab, setTab] = useState<'info' | 'logs'>('info')
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    setLoading(true)
    Promise.all([
      fetch(`/api/pods/${podName}?ns=${namespace}`).then(res => res.json()),
      fetch(`/api/pods/${podName}/logs?ns=${namespace}`).then(res => res.text())
    ]).then(([podData, logData]) => {
      setPod(podData)
      setLogs(logData)
    })
    .catch(console.error)
    .finally(() => setLoading(false))
  }, [podName, namespace])

  if (loading) return <div className="p-8 text-center">Loading pod details...</div>
  if (!pod) return <div className="p-8 text-center text-red-500">Pod not found</div>

  const age = new Date().getTime() - new Date(pod.metadata.creationTimestamp).getTime()
  const ageStr = age > 3600000 ? `${Math.floor(age/3600000)}h` : `${Math.floor(age/60000)}m`

  return (
    <div className="space-y-4">
      <button onClick={onBack} className="flex items-center gap-2 text-sm text-muted-foreground hover:text-foreground transition-colors">
        <ArrowLeft className="w-4 h-4" /> Back to Dashboard
      </button>

      <div className="border rounded-xl bg-card shadow-sm overflow-hidden">
        <div className="p-6 border-b">
          <div className="flex justify-between items-start">
            <div>
              <h1 className="text-2xl font-bold">{pod.metadata.name}</h1>
              <div className="text-muted-foreground text-sm mt-1">Namespace: {pod.metadata.namespace}</div>
            </div>
            <div className={cn(
              "px-3 py-1 rounded-full text-sm font-bold",
              pod.status.phase === 'Running' ? "bg-green-500/10 text-green-500" : "bg-yellow-500/10 text-yellow-500"
            )}>
              {pod.status.phase}
            </div>
          </div>
        </div>

        <div className="border-b bg-muted/30 px-6 flex gap-6">
          <button 
            onClick={() => setTab('info')}
            className={cn("py-3 text-sm font-medium border-b-2 transition-colors flex items-center gap-2", tab === 'info' ? "border-primary text-primary" : "border-transparent text-muted-foreground hover:text-foreground")}
          >
            <Info className="w-4 h-4" /> Info
          </button>
          <button 
            onClick={() => setTab('logs')}
            className={cn("py-3 text-sm font-medium border-b-2 transition-colors flex items-center gap-2", tab === 'logs' ? "border-primary text-primary" : "border-transparent text-muted-foreground hover:text-foreground")}
          >
            <Terminal className="w-4 h-4" /> Logs
          </button>
        </div>

        <div className="p-6">
          {tab === 'info' && (
            <div className="grid grid-cols-2 gap-6">
              <div className="space-y-4">
                <h3 className="font-semibold text-sm uppercase tracking-wider text-muted-foreground">Details</h3>
                <div className="grid grid-cols-[100px_1fr] gap-2 text-sm">
                  <div className="text-muted-foreground">Node</div>
                  <div className="font-mono flex items-center gap-2"><Cpu className="w-3 h-3" /> {pod.spec.nodeName}</div>
                  
                  <div className="text-muted-foreground">IP</div>
                  <div className="font-mono flex items-center gap-2"><Network className="w-3 h-3" /> {pod.status.podIP || 'N/A'}</div>

                  <div className="text-muted-foreground">Image</div>
                  <div className="font-mono flex items-center gap-2 break-all"><Box className="w-3 h-3 flex-shrink-0" /> {pod.spec.containers[0].image}</div>

                  <div className="text-muted-foreground">Age</div>
                  <div className="flex items-center gap-2"><Clock className="w-3 h-3" /> {ageStr}</div>
                  
                  <div className="text-muted-foreground">Restarts</div>
                  <div className="flex items-center gap-2"><RotateCw className="w-3 h-3" /> {pod.status.containerStatuses?.[0]?.restartCount || 0}</div>
                </div>
              </div>
            </div>
          )}

          {tab === 'logs' && (
            <div className="bg-black rounded-md p-4 font-mono text-xs text-gray-300 h-[400px] overflow-auto whitespace-pre-wrap">
              {logs || "No logs available."}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}