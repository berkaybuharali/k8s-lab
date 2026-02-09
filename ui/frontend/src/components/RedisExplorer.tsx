import { useEffect, useState } from 'react'
import { Search, RefreshCw, Trash2, Plus, Save, WifiOff } from 'lucide-react'
import redisLogo from '@/assets/redis_logo.svg'
import { cn } from '@/lib/utils'
import { usePanelLoading } from '../hooks/useLoadingTracker'

export function RedisExplorer({ isStale, refreshTrigger }: { isStale?: boolean; refreshTrigger?: number }) {
  const [keys, setKeys] = useState<string[]>([])
  const [loading, setLoading] = useState(true)
  const { setLoading: trackStart, setLoaded } = usePanelLoading('redis')
  const [pattern, setPattern] = useState('*')
  const [selectedKey, setSelectedKey] = useState<string | null>(null)
  const [value, setValue] = useState('')
  const [newKey, setNewKey] = useState('')
  const [newValue, setNewValue] = useState('')

  const fetchKeys = () => {
    setLoading(true)
    fetch(`/api/redis/keys?pattern=${pattern}`)
      .then(res => res.json())
      .then(data => setKeys(data || []))
      .catch(console.error)
      .finally(() => { setLoading(false); setLoaded() })
  }

  const fetchValue = (key: string) => {
    fetch(`/api/redis/get/${key}`)
      .then(res => res.text())
      .then(setValue)
      .catch(console.error)
  }

  const handleSet = () => {
    if (!newKey) return
    fetch('/api/redis/set', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ key: newKey, value: newValue })
    }).then(() => {
      setNewKey('')
      setNewValue('')
      fetchKeys()
    })
  }

  const handleDelete = (key: string) => {
    if (!confirm(`Delete key ${key}?`)) return
    fetch(`/api/redis/del/${key}`, { method: 'DELETE' })
      .then(() => {
        if (selectedKey === key) {
          setSelectedKey(null)
          setValue('')
        }
        fetchKeys()
      })
  }

  const handleFlush = () => {
    if (!confirm("Are you sure you want to FLUSH ALL DATA?")) return
    fetch('/api/redis/flush', { method: 'POST' }).then(fetchKeys)
  }

  useEffect(() => {
    trackStart()
    fetchKeys()
  }, [refreshTrigger])

  useEffect(() => {
    if (selectedKey) fetchValue(selectedKey)
  }, [selectedKey])

  return (
    <div className="border rounded-xl bg-card shadow-sm overflow-hidden flex flex-col h-[500px]">
      <div className="p-4 border-b flex justify-between items-center bg-muted/30">
        <h2 className="font-semibold flex items-center gap-2">
          <img src={redisLogo} alt="Redis" className="w-4 h-4" /> Redis Explorer
        </h2>
        <div className="flex gap-2">
          <button onClick={fetchKeys} className="p-1 hover:bg-muted rounded" title="Refresh">
            <RefreshCw className={cn("w-4 h-4", loading && "animate-spin")} />
          </button>
          <button onClick={handleFlush} className="p-1 hover:bg-red-100 text-red-500 rounded" title="Flush DB">
            <Trash2 className="w-4 h-4" />
          </button>
        </div>
      </div>

      <div className="flex flex-1 overflow-hidden">
        {/* Sidebar: Keys */}
        <div className="w-1/3 border-r flex flex-col bg-muted/10">
          <div className="p-2 border-b flex gap-2">
            <input 
              className="flex-1 text-xs border rounded px-2 py-1"
              value={pattern}
              onChange={e => setPattern(e.target.value)}
              placeholder="Pattern *"
            />
            <button onClick={fetchKeys} className="p-1 border rounded bg-background hover:bg-accent">
              <Search className="w-3 h-3" />
            </button>
          </div>
          <div className="flex-1 overflow-auto p-2 space-y-1">
            {keys.map(key => (
              <div 
                key={key} 
                onClick={() => setSelectedKey(key)}
                className={cn(
                  "px-2 py-1 text-sm rounded cursor-pointer truncate flex justify-between group",
                  selectedKey === key ? "bg-primary text-primary-foreground" : "hover:bg-accent"
                )}
              >
                <span>{key}</span>
                <button 
                  onClick={(e) => { e.stopPropagation(); handleDelete(key) }}
                  className="opacity-0 group-hover:opacity-100 hover:text-destructive"
                >
                  <Trash2 className="w-3 h-3" />
                </button>
              </div>
            ))}
            {keys.length === 0 && <div className="text-xs text-center text-muted-foreground mt-4">No keys found</div>}
          </div>
        </div>

        {/* Main: Value Editor */}
        <div className="flex-1 flex flex-col">
          {selectedKey ? (
            <div className="p-4 flex-1 flex flex-col gap-2">
              <div className="text-sm font-medium text-muted-foreground">Value for: <span className="text-foreground font-mono">{selectedKey}</span></div>
              <textarea 
                className="flex-1 border rounded p-2 font-mono text-sm bg-background resize-none focus:outline-none focus:ring-1 focus:ring-primary"
                value={value}
                readOnly 
              />
            </div>
          ) : (
            <div className="flex-1 flex items-center justify-center text-muted-foreground text-sm">
              Select a key to view value
            </div>
          )}

          {/* Add Key Footer */}
          <div className="p-3 border-t bg-muted/10 flex gap-2 items-center">
            <Plus className="w-4 h-4 text-muted-foreground" />
            <input 
              className="flex-1 text-xs border rounded px-2 py-1"
              placeholder="New Key"
              value={newKey}
              onChange={e => setNewKey(e.target.value)}
            />
            <input 
              className="flex-1 text-xs border rounded px-2 py-1"
              placeholder="Value"
              value={newValue}
              onChange={e => setNewValue(e.target.value)}
            />
            <button 
              onClick={handleSet}
              disabled={!newKey}
              className="p-1.5 bg-primary text-primary-foreground rounded hover:bg-primary/90 disabled:opacity-50"
            >
              <Save className="w-3 h-3" />
            </button>
          </div>
        </div>
      </div>
      
      {isStale && (
        <div className="px-4 py-2 bg-yellow-500/10 border-t border-yellow-500/20 text-yellow-600 text-xs flex items-center gap-2">
          <WifiOff className="w-3 h-3" /> Cannot reach Redis. Waiting for reconnection...
        </div>
      )}
    </div>
  )
}