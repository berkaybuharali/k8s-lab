import { useEffect, useState } from 'react'
import { cn } from "@/lib/utils"
import { Loader2 } from 'lucide-react'
import { CloudInfoPanel } from './components/CloudInfoPanel'
import { StatusPanel } from './components/StatusPanel'
import { ActionsPanel } from './components/ActionsPanel'
import { LogStream } from './components/LogStream'
import { NodesPanel } from './components/NodesPanel'
import { PodTable } from './components/PodTable'
import { PersistentDisks } from './components/PersistentDisks'
import { BackupList } from './components/BackupList'
import { Banner } from './components/Banner'
import { ThemeToggle } from './components/ThemeToggle'
import { PodDetail } from './components/PodDetail'
import { TerraformResources } from './components/TerraformResources'
import { RestoreDialog } from './components/RestoreDialog'
import { RedisExplorer } from './components/RedisExplorer'
import { useWebSocket } from './hooks/useWebSocket'
import { useApi } from './hooks/useApi'
import type { AuthStatus, GlobalStatus } from './types'

function App() {
  const [auth, setAuth] = useState<AuthStatus | null>(null)
  const [status, setStatus] = useState<GlobalStatus | null>(null)
  const [view, setView] = useState<'dashboard' | 'pod-detail' | 'tf-detail'>('dashboard')
  const [selectedPod, setSelectedPod] = useState<{name: string, ns: string} | null>(null)
  const [restoreOpen, setRestoreOpen] = useState(false)
  const [restoreBackup, setRestoreBackup] = useState('')
  
  const { logs, connected, isRunning, clearLogs } = useWebSocket()
  const { trigger } = useApi()

  useEffect(() => {
    fetch('/api/auth')
      .then(res => res.json())
      .then(data => setAuth(data))
      .catch(err => console.error("Failed to fetch auth:", err))

    fetchStatus()
    const interval = setInterval(fetchStatus, 15000)
    return () => clearInterval(interval)
  }, [])

  const fetchStatus = () => {
    fetch('/api/status')
      .then(res => res.json())
      .then(data => setStatus(data))
      .catch(err => console.error("Failed to fetch status:", err))
  }

  const handlePodClick = (name: string, ns: string) => {
    setSelectedPod({ name, ns })
    setView('pod-detail')
  }

  const handleRestoreClick = (name: string) => {
    setRestoreBackup(name)
    setRestoreOpen(true)
  }

  const handleRestoreAction = (backupName: string) => {
    trigger(`restore?backup=${backupName}&clean=true`)
  }

  const handleActionsRestore = () => {
    setRestoreBackup('')
    setRestoreOpen(true)
  }

  const isStale = status?.infra === 'Running' && status?.tunnel !== 'Connected'

  return (
    <div className="min-h-screen bg-background text-foreground flex flex-col">
      <header className="h-14 border-b px-4 flex items-center justify-between bg-card shrink-0 sticky top-0 z-10">
        <div className="flex items-center gap-4">
          <div className="flex items-center gap-2">
            <div className="w-8 h-8 bg-primary rounded flex items-center justify-center text-primary-foreground font-bold">
              K
            </div>
            <h1 className="font-semibold text-lg">Kubernetes Lab</h1>
          </div>

          {isRunning && (
            <div className="flex items-center gap-2 text-sm text-muted-foreground border-l pl-4">
              <Loader2 className="w-4 h-4 animate-spin text-primary" />
              <span>Operation in progress...</span>
            </div>
          )}
        </div>
        
        <div className="flex items-center gap-4">
          {status && (
            <div className={cn(
              "flex items-center gap-2 text-xs px-3 py-1 rounded-full border font-medium",
              status.tunnel === 'Connected' ? "bg-green-500/10 border-green-500/20 text-green-600" :
              status.tunnel === 'Reconnecting' ? "bg-yellow-500/10 border-yellow-500/20 text-yellow-600 animate-pulse" :
              "bg-red-500/10 border-red-500/20 text-red-600"
            )}>
              <div className={cn("w-2 h-2 rounded-full", 
                status.tunnel === 'Connected' ? "bg-green-500" :
                status.tunnel === 'Reconnecting' ? "bg-yellow-500" : "bg-red-500"
              )} />
              <span>Tunnel: {status.tunnel}</span>
            </div>
          )}

          {auth && (
            <div className={cn(
              "flex items-center gap-2 text-xs px-3 py-1 rounded-full border",
              auth.authenticated ? "bg-blue-500/10 border-blue-500/20 text-blue-600" : "bg-red-500/10 border-red-500/20 text-red-600"
            )}>
              <div className={cn("w-2 h-2 rounded-full", auth.authenticated ? "bg-blue-500" : "bg-red-500")} />
              <span>{auth.authenticated ? (auth.account || "Authenticated") : "Not Authenticated"}</span>
            </div>
          )}

          <ThemeToggle />
        </div>
      </header>

      <Banner auth={auth} status={status} />

      <div className="flex-1 flex overflow-hidden">
        <CloudInfoPanel auth={auth} onTFClick={() => setView('tf-detail')} />
        
        <main className="flex-1 overflow-auto p-6 space-y-6">
          {view === 'dashboard' && (
            <>
              <StatusPanel status={status} />
              
              <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                <LogStream 
                  logs={logs} 
                  connected={connected} 
                  onClear={clearLogs} 
                />
                <ActionsPanel 
                  auth={auth}
                  status={status} 
                  onTrigger={(op) => op === 'restore' ? handleActionsRestore() : trigger(op)} 
                  loading={isRunning}
                />
              </div>

              {status?.k8s === 'Ready' && (
                <div className={cn("space-y-6 transition-opacity", isStale && "opacity-60")}>
                  <NodesPanel isStale={isStale} />
                  <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                    <div className="space-y-6">
                      <PodTable isStale={isStale} onPodClick={handlePodClick} />
                      <RedisExplorer />
                    </div>
                    <div className="space-y-6">
                      <PersistentDisks isStale={isStale} />
                      <BackupList isStale={isStale} onRestore={handleRestoreClick} />
                    </div>
                  </div>
                </div>
              )}
            </>
          )}

          {view === 'pod-detail' && selectedPod && (
            <PodDetail 
              podName={selectedPod.name} 
              namespace={selectedPod.ns} 
              onBack={() => setView('dashboard')} 
            />
          )}

          {view === 'tf-detail' && (
            <TerraformResources onBack={() => setView('dashboard')} />
          )}
        </main>
      </div>

      <RestoreDialog 
        isOpen={restoreOpen} 
        onClose={() => setRestoreOpen(false)} 
        onRestore={handleRestoreAction}
        preSelectedBackup={restoreBackup}
      />
    </div>
  )
}

export default App
