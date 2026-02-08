import { useEffect, useRef, useState } from 'react'
import type { LogMessage } from '../types'

export function useWebSocket() {
  const [logs, setLogs] = useState<LogMessage[]>([])
  const [connected, setConnected] = useState(false)
  const [isRunning, setIsRunning] = useState(false)
  const ws = useRef<WebSocket | null>(null)
  const reconnectAttempts = useRef(0)

  useEffect(() => {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const host = window.location.host
    const wsUrl = `${protocol}//${host}/ws/logs`
    
    function connect() {
      ws.current = new WebSocket(wsUrl)

      ws.current.onopen = () => {
        setConnected(true)
        reconnectAttempts.current = 0
        console.log('WS connected')
      }

      ws.current.onclose = () => {
        setConnected(false)
        if (reconnectAttempts.current < 10) {
          reconnectAttempts.current++
          console.log(`WS disconnected, reconnecting in 3s (attempt ${reconnectAttempts.current}/10)...`)
          setTimeout(connect, 3000)
        } else {
          console.error('WS reconnection failed after 10 attempts')
        }
      }

      ws.current.onmessage = (event) => {
        try {
          const msg: LogMessage = JSON.parse(event.data)
          setLogs(prev => [...prev, msg])
          
          if (msg.type === 'start') setIsRunning(true)
          if (msg.type === 'done' || msg.type === 'error') setIsRunning(false)
        } catch (e) {
          console.error('Failed to parse WS message:', e)
        }
      }
    }

    connect()

    return () => {
      // Prevent reconnect on unmount
      reconnectAttempts.current = 100 
      ws.current?.close()
    }
  }, [])

  const clearLogs = () => setLogs([])

  return { logs, connected, isRunning, clearLogs }
}