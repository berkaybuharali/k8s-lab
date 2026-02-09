import { useEffect, useState } from 'react'
import { ArrowLeft, Terminal, Info, RotateCw, Cpu, Clock, Box, Network, FileCode } from 'lucide-react'
import type { K8sPod } from '../types'
import { cn } from '@/lib/utils'

interface PodDetailProps {
  podName: string
  namespace: string
  onBack: () => void
}

type Tab = 'info' | 'logs' | 'deployment' | 'service'

export function PodDetail({ podName, namespace, onBack }: PodDetailProps) {
  const [pod, setPod] = useState<K8sPod | null>(null)
  const [logs, setLogs] = useState<string>('')
  const [deploymentYaml, setDeploymentYaml] = useState<string | null>(null)
  const [serviceYaml, setServiceYaml] = useState<string | null>(null)
  const [tab, setTab] = useState<Tab>('info')
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    setLoading(true)
    Promise.all([
      fetch(`/api/pods/${podName}?ns=${namespace}`).then(res => res.json()),
      fetch(`/api/pods/${podName}/logs?ns=${namespace}`).then(res => res.text()),
      fetch(`/api/pods/${podName}/deployment?ns=${namespace}`).then(res => res.ok ? res.text() : null),
      fetch(`/api/pods/${podName}/service?ns=${namespace}`).then(res => res.ok ? res.text() : null),
    ]).then(([podData, logData, deployData, svcData]) => {
      setPod(podData)
      setLogs(logData)
      setDeploymentYaml(deployData)
      setServiceYaml(svcData)
    })
    .catch(console.error)
    .finally(() => setLoading(false))
  }, [podName, namespace])

  if (loading) return <div className="p-8 text-center">Loading pod details...</div>
  if (!pod) return <div className="p-8 text-center text-red-500">Pod not found</div>

  const age = new Date().getTime() - new Date(pod.metadata.creationTimestamp).getTime()
  const ageStr = age > 3600000 ? `${Math.floor(age/3600000)}h` : `${Math.floor(age/60000)}m`

  const tabs: { id: Tab; label: string; icon: typeof Info; available: boolean }[] = [
    { id: 'info', label: 'Info', icon: Info, available: true },
    { id: 'logs', label: 'Logs', icon: Terminal, available: true },
    { id: 'deployment', label: 'Deployment', icon: FileCode, available: deploymentYaml !== null },
    { id: 'service', label: 'Service', icon: FileCode, available: serviceYaml !== null },
  ]

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
          {tabs.filter(t => t.available).map(t => (
            <button
              key={t.id}
              onClick={() => setTab(t.id)}
              className={cn("py-3 text-sm font-medium border-b-2 transition-colors flex items-center gap-2",
                tab === t.id ? "border-primary text-primary" : "border-transparent text-muted-foreground hover:text-foreground"
              )}
            >
              <t.icon className="w-4 h-4" /> {t.label}
            </button>
          ))}
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

          {tab === 'deployment' && deploymentYaml && (
            <div className="bg-black rounded-md p-4 font-mono text-xs text-gray-300 h-[400px] overflow-auto whitespace-pre-wrap">
              {deploymentYaml}
            </div>
          )}

          {tab === 'service' && serviceYaml && (
            <div className="bg-black rounded-md p-4 font-mono text-xs text-gray-300 h-[400px] overflow-auto whitespace-pre-wrap">
              {serviceYaml}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
