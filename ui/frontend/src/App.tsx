import { useEffect, useRef, useState } from 'react'
import { cn } from "@/lib/utils"
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
import { PodDetail } from './components/PodDetail'
import { TerraformResources } from './components/TerraformResources'
import { RestoreDialog } from './components/RestoreDialog'
import { RedisExplorer } from './components/RedisExplorer'
import { DiskSnapshots } from './components/DiskSnapshots'
import { AboutPage } from './components/AboutPage'
import { ShopPage } from './components/ShopPage'
import { BackofficePage } from './components/BackofficePage'
import { Layout } from './components/Layout'
import { useWebSocket } from './hooks/useWebSocket'
import { useApi } from './hooks/useApi'
import { useAppStatus } from './hooks/useAppStatus'

function AppInner() {
  const { auth, status, initialLoading, fetchStatus } = useAppStatus()

  const [view, setView] = useState<'dashboard' | 'pod-detail' | 'tf-detail' | 'about' | 'architecture' | 'shop' | 'backoffice'>('dashboard')
  const [selectedPod, setSelectedPod] = useState<{name: string, ns: string} | null>(null)
  const [restoreOpen, setRestoreOpen] = useState(false)
  const [restoreBackup, setRestoreBackup] = useState('')
  const [opDoneCounter, setOpDoneCounter] = useState(0)

  const { logs, connected, isRunning, clearLogs } = useWebSocket()
  const { trigger } = useApi()

  const { setLoading: trackStatusStart, setLoaded: trackStatusDone } = usePanelLoading('cluster-status', 'Connecting to cluster...')
  const statusTrackedRef = useRef(false)

  useEffect(() => {
    if (!initialLoading && !statusTrackedRef.current) {
      statusTrackedRef.current = true
      trackStatusStart()
    }
  }, [initialLoading])

  useEffect(() => {
    if (status?.k8s === 'Ready') {
      trackStatusDone()
    }
  }, [status?.k8s])

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
    <Layout view={view} setView={setView} auth={auth} status={status} isRunning={isRunning}>
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
                <LogStream logs={logs} connected={connected} onClear={clearLogs} />
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
            <PodDetail podName={selectedPod.name} namespace={selectedPod.ns} onBack={() => setView('dashboard')} />
          )}

          {view === 'tf-detail' && (
            <TerraformResources onBack={() => setView('dashboard')} />
          )}

          {view === 'architecture' && (
            <div className="w-full h-full">
              <iframe src="/architecture.html" className="w-full h-full border-0" title="Architecture" />
            </div>
          )}

          {view === 'shop' && <ShopPage />}

          {view === 'backoffice' && <BackofficePage />}

          {view === 'about' && <AboutPage onBack={() => setView('dashboard')} />}
        </main>
      </div>

      <RestoreDialog
        isOpen={restoreOpen}
        onClose={() => setRestoreOpen(false)}
        onRestore={handleRestoreAction}
        preSelectedBackup={restoreBackup}
      />
    </Layout>
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
