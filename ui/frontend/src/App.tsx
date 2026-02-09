import { useEffect, useRef, useState } from 'react'
import { cn } from "@/lib/utils"
import { Loader2 } from 'lucide-react'
import { LoadingTrackerProvider, usePanelLoading } from './hooks/useLoadingTracker'
import { LoadingScreen } from './components/LoadingScreen'
import { GlobalLoadingBar } from './components/GlobalLoadingBar'
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
import { DiskSnapshots } from './components/DiskSnapshots'
import { AboutPage } from './components/AboutPage'
import { useWebSocket } from './hooks/useWebSocket'
import { useApi } from './hooks/useApi'
import type { AuthStatus, GlobalStatus } from './types'

function AppInner() {
  const [auth, setAuth] = useState<AuthStatus | null>(null)
  const [status, setStatus] = useState<GlobalStatus | null>(null)
  const [view, setView] = useState<'dashboard' | 'pod-detail' | 'tf-detail' | 'about'>('dashboard')
  const [selectedPod, setSelectedPod] = useState<{name: string, ns: string} | null>(null)
  const [restoreOpen, setRestoreOpen] = useState(false)
  const [restoreBackup, setRestoreBackup] = useState('')
  const [opDoneCounter, setOpDoneCounter] = useState(0)
  const [initialLoading, setInitialLoading] = useState(true)

  const { logs, connected, isRunning, clearLogs } = useWebSocket()
  const { trigger } = useApi()

  // Track "waiting for cluster status" as a loading item.
  // This bridges the gap between the splash screen and panels mounting.
  const { setLoading: trackStatusStart, setLoaded: trackStatusDone } = usePanelLoading('cluster-status', 'Connecting to cluster...')
  const statusTrackedRef = useRef(false)

  // Start tracking right away (before panels mount)
  useEffect(() => {
    if (!initialLoading && !statusTrackedRef.current) {
      statusTrackedRef.current = true
      trackStatusStart()
    }
  }, [initialLoading])

  // Mark cluster-status as loaded when k8s becomes Ready
  useEffect(() => {
    if (status?.k8s === 'Ready') {
      trackStatusDone()
    }
  }, [status?.k8s])

  // Refresh status when an operation completes (done/error)
  useEffect(() => {
    if (logs.length > 0) {
      const last = logs[logs.length - 1]
      if (last.type === 'done' || last.type === 'error') {
        setOpDoneCounter(c => c + 1)
        fetchStatus()
        const t1 = setTimeout(fetchStatus, 5000)
        const t2 = setTimeout(fetchStatus, 15000)
        return () => { clearTimeout(t1); clearTimeout(t2) }
      }
    }
  }, [logs.length])

  useEffect(() => {
    Promise.all([
      fetch('/api/auth').then(res => res.json()).then(data => setAuth(data)),
      fetch('/api/status').then(res => res.json()).then(data => setStatus(data)),
    ]).catch(err => console.error("Failed to fetch initial data:", err))
      .finally(() => setInitialLoading(false))

    // Poll every 2s at startup so data appears fast, then relax to 10s
    let intervalId = setInterval(fetchStatus, 2000)
    const switchTimer = setTimeout(() => {
      clearInterval(intervalId)
      intervalId = setInterval(fetchStatus, 10000)
    }, 30000)

    return () => {
      clearInterval(intervalId)
      clearTimeout(switchTimer)
    }
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
    <div className="h-screen bg-background text-foreground flex flex-col overflow-hidden">
      <header className="h-14 border-b px-4 flex items-center justify-between bg-card shrink-0 sticky top-0 z-10">
        <div className="flex items-center gap-4">
          <div className="flex items-center gap-2">
            <button onClick={() => setView('dashboard')} className="font-bold text-lg tracking-tight hover:opacity-80 transition-opacity">
              k8s<span className="text-primary">-lab</span>
            </button>
            <button onClick={() => setView('about')} className="text-xs text-muted-foreground hover:text-foreground transition-colors ml-2">
              About
            </button>
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
              status.tunnel === 'Starting' ? "bg-yellow-500/10 border-yellow-500/20 text-yellow-600" :
              (status.tunnel === 'Idle' || status.tunnel === 'N/A') ? "border-border text-muted-foreground" :
              "bg-red-500/10 border-red-500/20 text-red-600"
            )}>
              <div className={cn("w-2 h-2 rounded-full",
                status.tunnel === 'Connected' ? "bg-green-500" :
                status.tunnel === 'Reconnecting' || status.tunnel === 'Starting' ? "bg-yellow-500" :
                (status.tunnel === 'Idle' || status.tunnel === 'N/A') ? "bg-muted-foreground" :
                "bg-red-500"
              )} />
              <span>Tunnel: {status.tunnel}</span>
            </div>
          )}

          {auth && (
            <div className={cn(
              "flex items-center gap-2 text-xs px-3 py-1 rounded-full border",
              auth.authenticated ? "bg-green-500/10 border-green-500/20 text-green-600" : "bg-red-500/10 border-red-500/20 text-red-600"
            )}>
              <div className={cn("w-2 h-2 rounded-full", auth.authenticated ? "bg-green-500" : "bg-red-500")} />
              <span>{auth.authenticated ? "Authenticated" : "Not Authenticated"}</span>
            </div>
          )}

          <ThemeToggle />
        </div>
      </header>

      <Banner auth={auth} status={status} />

      <LoadingScreen ready={!initialLoading} />

      <GlobalLoadingBar />

      <div className="flex-1 flex overflow-hidden">
        <CloudInfoPanel auth={auth} onTFClick={() => setView('tf-detail')} />

        <main className="flex-1 overflow-auto p-6 space-y-6">
          {view === 'dashboard' && (
            <>
              <StatusPanel status={status} isRunning={isRunning} provider={auth?.provider} runningOp={
                [...logs].reverse().find((l: { type: string }) => l.type === 'start')?.data.match(/operation: (.+)/)?.[1] || ''
              } />

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
                  refreshTrigger={opDoneCounter}
                />
              </div>

              {status?.k8s === 'Ready' && (
                <div className={cn("space-y-6 transition-opacity", isStale && "opacity-60")}>
                  <NodesPanel isStale={isStale} provider={auth?.provider} />
                  <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                    <div className="space-y-6">
                      <PodTable isStale={isStale} onPodClick={handlePodClick} />
                      <RedisExplorer refreshTrigger={opDoneCounter} />
                    </div>
                    <div className="space-y-6">
                      <PersistentDisks isStale={isStale} />
                      <BackupList isStale={isStale} onRestore={handleRestoreClick} refreshTrigger={opDoneCounter} />
                      <DiskSnapshots isStale={isStale} provider={auth?.provider} refreshTrigger={opDoneCounter} />
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

          {view === 'about' && (
            <AboutPage onBack={() => setView('dashboard')} />
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

function App() {
  return (
    <LoadingTrackerProvider>
      <AppInner />
    </LoadingTrackerProvider>
  )
}

export default App
