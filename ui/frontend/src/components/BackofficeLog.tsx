import { useEffect, useState } from 'react'
import { Activity } from 'lucide-react'

interface Log {
    timestamp: string
    system: string
    message: string
}

export function BackofficeLog() {
  const [logs, setLogs] = useState<Log[]>([])

  useEffect(() => {
    fetch('/api/agent/activity')
      .then(res => res.json())
      .then(setLogs)
      .catch(console.error)
  }, [])

  return (
    <div className="border rounded-xl bg-card shadow-sm overflow-hidden h-full flex flex-col">
      <div className="px-5 py-3 border-b flex items-center gap-2">
        <Activity className="w-4 h-4 text-muted-foreground" />
        <h2 className="text-sm font-semibold">Agent Activity</h2>
      </div>
      <div className="flex-1 overflow-y-auto p-0">
          <div className="divide-y">
            {logs.length === 0 ? (
                <div className="p-4 text-xs text-muted-foreground text-center">No recent activity</div>
            ) : (
                logs.map((log, i) => (
                    <div key={i} className="p-3 hover:bg-muted/50 transition-colors text-xs">
                        <div className="flex justify-between text-muted-foreground mb-1">
                            <span className="capitalize font-medium text-foreground">{log.system}</span>
                            <span className="opacity-70">{new Date(log.timestamp).toLocaleTimeString()}</span>
                        </div>
                        <div className="line-clamp-2">{log.message}</div>
                    </div>
                ))
            )}
          </div>
      </div>
    </div>
  )
}
