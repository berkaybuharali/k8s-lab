import { useEffect, useState } from 'react'
import { Trash, Database, Server, AppWindow, Rocket, Archive, RotateCcw, Check, Layers, Bot } from 'lucide-react'
import type { AuthStatus, GlobalStatus } from '../types'
import { cn } from '@/lib/utils'

interface ActionsPanelProps {
  auth: AuthStatus | null
  status: GlobalStatus | null
  onTrigger: (op: string) => void
  loading: boolean
  refreshTrigger?: number
}

export function ActionsPanel({ auth, status, onTrigger, loading, refreshTrigger }: ActionsPanelProps) {
  const [redisHasData, setRedisHasData] = useState(false)
  const notAuth = !!(auth && !auth.authenticated)
  const infraReady = status?.infra === 'Running'
  const k8sReady = status?.k8s === 'Ready'
  const toolsReady = status?.tools === 'Installed'
  const appsReady = status?.apps === 'Deployed'

  useEffect(() => {
    if (!appsReady) return
    fetch('/api/redis/dbsize')
      .then(res => res.json())
      .then(data => setRedisHasData(data.count > 0))
      .catch(() => setRedisHasData(false))
  }, [appsReady, refreshTrigger])

  const getStepStatus = (step: number, _id: string) => {
    if (step === 1 && infraReady) return 'done'
    if (step === 2 && toolsReady) return 'done'
    if (step === 3 && appsReady) return 'done'
    if (step === 4 && appsReady && redisHasData) return 'done' // Assume agents are deployed if apps are ready & data is seeded? No perfect check yet.
    if (step === 5 && redisHasData) return 'done'
    return 'pending'
  }

  const steps = [
    { id: 'deploy-infra', label: 'Deploy Infra', icon: Server, disabled: loading || notAuth, step: 1 },
    { id: 'deploy-tools', label: 'Deploy Tools', icon: Layers, disabled: loading || notAuth || !k8sReady, step: 2 },
    { id: 'deploy-applications', label: 'Deploy Apps', icon: AppWindow, disabled: loading || notAuth || !toolsReady, step: 3 },
    { id: 'deploy-agents', label: 'Deploy Agents', icon: Bot, disabled: loading || notAuth || !appsReady, step: 4 },
    { id: 'seed-data', label: 'Seed Data', icon: Database, disabled: loading || notAuth || !appsReady, step: 5 },
  ]

  return (
    <div className="border rounded-xl shadow-sm bg-card text-card-foreground flex flex-col overflow-hidden">
      <div className="px-5 py-3 border-b">
        <h2 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">Actions</h2>
      </div>
      <div className="p-5 flex flex-col gap-4">
        {/* Quick Actions */}
        <div>
          <div className="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground mb-2">Quick</div>
          <div className="flex gap-2">
            <button
              onClick={() => onTrigger('deploy')}
              disabled={loading || notAuth}
              className={cn(
                "flex-1 flex items-center justify-center gap-2 py-2 px-3 rounded-lg border bg-primary text-primary-foreground text-sm font-medium transition-all hover:bg-primary/90",
                (loading || notAuth) && "opacity-40 cursor-not-allowed"
              )}
            >
              <Rocket className="w-3.5 h-3.5" /> Deploy All
            </button>
            <button
              onClick={() => onTrigger('destroy')}
              disabled={loading || notAuth}
              className={cn(
                "flex-1 flex items-center justify-center gap-2 py-2 px-3 rounded-lg border border-destructive/30 text-destructive text-sm font-medium transition-all hover:bg-destructive/10",
                (loading || notAuth) && "opacity-40 cursor-not-allowed"
              )}
            >
              <Trash className="w-3.5 h-3.5" /> Destroy All
            </button>
          </div>
        </div>

        {/* Step by Step */}
        <div>
          <div className="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground mb-2">Step by Step</div>
          <div className="flex flex-wrap gap-2">
            {steps.map((action) => {
              const isDone = getStepStatus(action.step, action.id) === 'done'
              return (
                <button
                  key={action.id}
                  onClick={() => onTrigger(action.id)}
                  disabled={action.disabled}
                  className={cn(
                    "flex items-center gap-2 py-1.5 px-3 rounded-lg border text-sm font-medium transition-all",
                    action.disabled
                      ? "opacity-40 cursor-not-allowed bg-muted"
                      : "hover:bg-accent hover:text-accent-foreground"
                  )}
                >
                  <div className={cn(
                    "w-5 h-5 rounded-full flex items-center justify-center text-[10px] font-bold border flex-shrink-0",
                    isDone ? "bg-green-500 border-green-500 text-white" :
                    "bg-background border-muted-foreground/40 text-muted-foreground"
                  )}>
                    {isDone ? <Check className="w-3 h-3" /> : action.step}
                  </div>
                  {action.label}
                </button>
              )
            })}
          </div>
        </div>

        {/* Backup & Restore */}
        <div>
          <div className="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground mb-2">Backup & Restore</div>
          <div className="flex gap-2">
            <button
              onClick={() => onTrigger('backup')}
              disabled={loading || notAuth || !appsReady}
              className={cn(
                "flex items-center gap-2 py-1.5 px-3 rounded-lg border text-sm transition-colors hover:bg-accent",
                (loading || notAuth || !appsReady) && "opacity-40 cursor-not-allowed"
              )}
            >
              <Archive className="w-3.5 h-3.5" /> Create Backup
            </button>
            <button
              onClick={() => onTrigger('restore')}
              disabled={loading || notAuth || !toolsReady}
              className={cn(
                "flex items-center gap-2 py-1.5 px-3 rounded-lg border border-green-500/30 text-green-600 text-sm font-medium transition-colors hover:bg-green-500/10",
                (loading || notAuth || !toolsReady) && "opacity-40 cursor-not-allowed"
              )}
            >
              <RotateCcw className="w-3.5 h-3.5" /> Restore from Backup
            </button>
          </div>
        </div>
      </div>

      {notAuth && (
        <div className="px-4 py-2 bg-destructive/10 border-t border-destructive/20 text-destructive text-xs text-center font-medium">
          Authentication required to perform any operations
        </div>
      )}
    </div>
  )
}
