import { useEffect, useState } from 'react'
import { cn } from "@/lib/utils"
import { CloudInfoPanel } from './components/CloudInfoPanel'
import { StatusPanel } from './components/StatusPanel'
import { ActionsPanel } from './components/ActionsPanel'
import { LogStream } from './components/LogStream'
import { useWebSocket } from './hooks/useWebSocket'
import { useApi } from './hooks/useApi'
import type { AuthStatus, GlobalStatus } from './types'

function App() {
  const [auth, setAuth] = useState<AuthStatus | null>(null)
  const [status, setStatus] = useState<GlobalStatus | null>(null)
  
  const { logs, connected, isRunning, clearLogs } = useWebSocket()
  const { trigger } = useApi()

  useEffect(() => {
    // Initial fetch
    fetch('/api/auth')
      .then(res => res.json())
      .then(data => setAuth(data))
      .catch(err => console.error("Failed to fetch auth:", err))

    fetchStatus()

    // Poll status every 15s
    const interval = setInterval(fetchStatus, 15000)
    return () => clearInterval(interval)
  }, [])

  const fetchStatus = () => {
    fetch('/api/status')
      .then(res => res.json())
      .then(data => setStatus(data))
      .catch(err => console.error("Failed to fetch status:", err))
  }

  return (
    <div className="min-h-screen bg-background text-foreground flex flex-col">
      {/* Header */}
      <header className="h-14 border-b px-4 flex items-center justify-between bg-card shrink-0">
        <div className="flex items-center gap-2">
          <div className="w-8 h-8 bg-primary rounded flex items-center justify-center text-primary-foreground font-bold">
            K
          </div>
          <h1 className="font-semibold text-lg">Kubernetes Lab</h1>
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
        </div>
      </header>

      {/* Main Layout */}
      <div className="flex-1 flex overflow-hidden">
        <CloudInfoPanel auth={auth} />
        
        <main className="flex-1 overflow-auto p-6 space-y-6">
          <StatusPanel status={status} />
          
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            <LogStream 
              logs={logs} 
              connected={connected} 
              onClear={clearLogs} 
            />
            
            <ActionsPanel 
              status={status} 
              onTrigger={trigger} 
              loading={isRunning}
            />
          </div>
        </main>
      </div>
    </div>
  )
}

export default App