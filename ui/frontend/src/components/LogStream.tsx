import { useEffect, useRef } from 'react'
import type { LogMessage } from '../types'
import { cn } from '@/lib/utils'
import { Trash2, Wifi, WifiOff } from 'lucide-react'

interface LogStreamProps {
  logs: LogMessage[]
  connected: boolean
  onClear: () => void
}

export function LogStream({ logs, connected, onClear }: LogStreamProps) {
  const bottomRef = useRef<HTMLDivElement>(null)

  // Auto-scroll
  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [logs])

  return (
    <div className="border rounded-lg shadow-sm bg-card text-card-foreground flex flex-col h-[400px]">
      <div className="p-4 border-b flex items-center justify-between bg-muted/30">
        <div className="flex items-center gap-2">
          <h2 className="font-semibold">Operations Log</h2>
          {connected ? (
            <Wifi className="w-4 h-4 text-green-500" />
          ) : (
            <WifiOff className="w-4 h-4 text-red-500" />
          )}
        </div>
        <button 
          onClick={onClear}
          className="text-muted-foreground hover:text-destructive transition-colors"
          title="Clear Logs"
        >
          <Trash2 className="w-4 h-4" />
        </button>
      </div>
      
      <div className="flex-1 bg-black p-4 font-mono text-xs overflow-auto">
        {logs.length === 0 ? (
          <p className="text-gray-500">&gt; Ready for operations...</p>
        ) : (
          logs.map((log, i) => (
            <div key={i} className={cn(
              "whitespace-pre-wrap break-all mb-1",
              log.type === 'error' ? "text-red-400" :
              log.type === 'start' ? "text-blue-400 font-bold" :
              log.type === 'done' ? "text-green-400 font-bold" :
              "text-gray-300"
            )}>
              {log.data}
            </div>
          ))
        )}
        <div ref={bottomRef} />
      </div>
    </div>
  )
}
