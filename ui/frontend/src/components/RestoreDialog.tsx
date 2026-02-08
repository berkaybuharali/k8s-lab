import { useEffect, useState } from 'react'
import { X, Archive, AlertTriangle, Check, RotateCcw } from 'lucide-react'
import type { VeleroBackup } from '../types'
import { cn } from '@/lib/utils'

interface RestoreDialogProps {
  isOpen: boolean
  onClose: () => void
  onRestore: (backupName: string) => void
  preSelectedBackup?: string
}

export function RestoreDialog({ isOpen, onClose, onRestore, preSelectedBackup }: RestoreDialogProps) {
  const [backups, setBackups] = useState<VeleroBackup[]>([])
  const [selected, setSelected] = useState<string>(preSelectedBackup || '')
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    if (isOpen) {
      setLoading(true)
      fetch('/api/backups')
        .then(res => res.json())
        .then(data => {
          setBackups(data.items || [])
          if (!selected && data.items?.length > 0) {
            setSelected(data.items[0].metadata.name)
          }
        })
        .finally(() => setLoading(false))
    }
  }, [isOpen])

  useEffect(() => {
    if (preSelectedBackup) setSelected(preSelectedBackup)
  }, [preSelectedBackup])

  if (!isOpen) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm">
      <div className="bg-background border rounded-lg shadow-lg w-full max-w-lg overflow-hidden animate-in fade-in zoom-in-95 duration-200">
        <div className="p-4 border-b flex justify-between items-center">
          <h2 className="font-semibold text-lg flex items-center gap-2">
            <RotateCcw className="w-5 h-5 text-blue-500" /> Restore Application
          </h2>
          <button onClick={onClose} className="text-muted-foreground hover:text-foreground">
            <X className="w-5 h-5" />
          </button>
        </div>

        <div className="p-6 space-y-4">
          <div className="bg-yellow-500/10 border border-yellow-500/20 rounded p-3 flex gap-3 text-sm text-yellow-600">
            <AlertTriangle className="w-5 h-5 shrink-0" />
            <div>
              <p className="font-medium">Warning: Destructive Action</p>
              <p>Restoring will overwrite existing data in the application namespace.</p>
            </div>
          </div>

          <div className="space-y-2">
            <label className="text-sm font-medium">Select Backup</label>
            {loading ? (
              <div className="p-4 text-center text-sm text-muted-foreground">Loading backups...</div>
            ) : backups.length === 0 ? (
              <div className="p-4 border rounded text-center text-muted-foreground text-sm">No backups found.</div>
            ) : (
              <div className="border rounded-md divide-y max-h-[200px] overflow-auto">
                {backups.map(b => (
                  <button
                    key={b.metadata.name}
                    onClick={() => setSelected(b.metadata.name)}
                    className={cn(
                      "w-full flex items-center justify-between p-3 text-sm hover:bg-accent transition-colors",
                      selected === b.metadata.name && "bg-accent"
                    )}
                  >
                    <div className="flex items-center gap-3">
                      <div className={cn(
                        "w-4 h-4 rounded-full border flex items-center justify-center",
                        selected === b.metadata.name ? "border-primary bg-primary text-primary-foreground" : "border-muted-foreground"
                      )}>
                        {selected === b.metadata.name && <Check className="w-3 h-3" />}
                      </div>
                      <div className="flex flex-col text-left">
                        <span className="font-medium">{b.metadata.name}</span>
                        <span className="text-xs text-muted-foreground">{new Date(b.status.completionTimestamp).toLocaleString()}</span>
                      </div>
                    </div>
                    <Archive className="w-4 h-4 text-muted-foreground" />
                  </button>
                ))}
              </div>
            )}
          </div>
        </div>

        <div className="p-4 border-t bg-muted/30 flex justify-end gap-3">
          <button onClick={onClose} className="px-4 py-2 text-sm font-medium hover:underline">Cancel</button>
          <button 
            onClick={() => {
              onRestore(selected)
              onClose()
            }}
            disabled={!selected}
            className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-md text-sm font-medium disabled:opacity-50"
          >
            Start Restore
          </button>
        </div>
      </div>
    </div>
  )
}
