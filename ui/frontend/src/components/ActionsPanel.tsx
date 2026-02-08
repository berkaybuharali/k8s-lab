import { Trash, Layers, Database, Server, AppWindow, Rocket, Archive, RotateCcw, Check } from 'lucide-react'
import type { AuthStatus, GlobalStatus } from '../types'
import { cn } from '@/lib/utils'

interface ActionsPanelProps {
  auth: AuthStatus | null
  status: GlobalStatus | null
  onTrigger: (op: string) => void
  loading: boolean
}

export function ActionsPanel({ auth, status, onTrigger, loading }: ActionsPanelProps) {
  const notAuth = !!(auth && !auth.authenticated)
  const infraReady = status?.infra === 'Running'
  const k8sReady = status?.k8s === 'Ready'
  const toolsReady = status?.tools === 'Installed'
  const appsReady = status?.apps === 'Deployed'

  // Step Status Helper
  const getStepStatus = (step: number) => {
    if (step === 1 && infraReady) return 'done'
    if (step === 2 && toolsReady) return 'done'
    if (step === 3 && appsReady) return 'done'
    return 'pending'
  }

  const steps = [
    {
      id: 'deploy-infra',
      label: 'Deploy Infrastructure',
      icon: Server,
      disabled: loading || notAuth,
      desc: "Terraform + Talos Bootstrap",
      step: 1
    },
    {
      id: 'deploy-tools',
      label: 'Deploy Platform Tools',
      icon: Layers,
      disabled: loading || notAuth || !k8sReady,
      desc: "CSI Driver, Velero",
      step: 2
    },
    {
      id: 'deploy-applications',
      label: 'Deploy Applications',
      icon: AppWindow,
      disabled: loading || notAuth || !toolsReady,
      desc: "NGINX, Redis",
      step: 3
    },
    {
      id: 'seed-redis',
      label: 'Seed Test Data',
      icon: Database,
      disabled: loading || notAuth || !appsReady,
      desc: "Fill Redis with data",
      step: 4
    }
  ]

  return (
    <div className="p-0 border rounded-lg shadow-sm bg-card text-card-foreground flex flex-col overflow-hidden">
      <div className="p-6 flex flex-col gap-6">
        {/* Quick Actions */}
        <div>
          <h2 className="font-semibold mb-3 text-sm uppercase tracking-wider text-muted-foreground">Quick Actions</h2>
          <div className="grid grid-cols-2 gap-3">
            <button
              onClick={() => onTrigger('deploy')}
              disabled={loading || notAuth}
              className={cn(
                "flex items-center justify-center gap-2 p-3 rounded-md border bg-primary text-primary-foreground font-medium transition-all hover:bg-primary/90",
                (loading || notAuth) && "opacity-50 cursor-not-allowed"
              )}
            >
              <Rocket className="w-4 h-4" /> Deploy All
            </button>
            
            <button
              onClick={() => onTrigger('destroy')}
              disabled={loading || notAuth}
              className={cn(
                "flex items-center justify-center gap-2 p-3 rounded-md border border-destructive/20 text-destructive font-medium transition-all hover:bg-destructive/10",
                (loading || notAuth) && "opacity-50 cursor-not-allowed"
              )}
            >
              <Trash className="w-4 h-4" /> Destroy All
            </button>
          </div>
        </div>

        <div className="border-t" />

        {/* Step by Step */}
        <div>
          <h2 className="font-semibold mb-3 text-sm uppercase tracking-wider text-muted-foreground">Step by Step</h2>
          <div className="space-y-2">
            {steps.map((action) => {
              const isDone = getStepStatus(action.step) === 'done'
              return (
                <button
                  key={action.id}
                  onClick={() => onTrigger(action.id)}
                  disabled={action.disabled}
                  className={cn(
                    "w-full flex items-center gap-3 p-2 rounded-md border transition-all text-left group",
                    action.disabled 
                      ? "opacity-50 cursor-not-allowed bg-muted" 
                      : "hover:bg-accent hover:text-accent-foreground"
                  )}
                >
                  {/* Step Badge */}
                  <div className={cn(
                    "w-8 h-8 rounded-full flex items-center justify-center text-xs font-bold border transition-colors",
                    isDone ? "bg-green-500 border-green-500 text-white" : 
                    action.disabled ? "bg-muted border-muted-foreground/30 text-muted-foreground" :
                    "bg-background border-primary text-primary group-hover:bg-primary group-hover:text-primary-foreground"
                  )}>
                    {isDone ? <Check className="w-4 h-4" /> : action.step}
                  </div>

                  <div className="flex-1">
                    <div className="font-medium text-sm flex items-center gap-2">
                      {action.label}
                    </div>
                    <div className="text-xs text-muted-foreground">{action.desc}</div>
                  </div>

                  <action.icon className="w-4 h-4 text-muted-foreground opacity-50 group-hover:opacity-100" />
                </button>
              )
            })}
          </div>
        </div>

        <div className="border-t" />

        {/* Backup & Restore */}
        <div>
          <h2 className="font-semibold mb-3 text-sm uppercase tracking-wider text-muted-foreground">Disaster Recovery</h2>
          <div className="grid grid-cols-2 gap-3">
            <button
              onClick={() => onTrigger('backup')}
              disabled={loading || notAuth || !appsReady}
              className={cn(
                "flex items-center justify-center gap-2 p-2 rounded-md border hover:bg-accent transition-colors text-sm",
                (loading || notAuth || !appsReady) && "opacity-50 cursor-not-allowed"
              )}
            >
              <Archive className="w-4 h-4" /> Create Backup
            </button>
            
            <button
              onClick={() => onTrigger('restore')} 
              disabled={loading || notAuth || !toolsReady} 
              className={cn(
                "flex items-center justify-center gap-2 p-2 rounded-md border hover:bg-accent transition-colors text-sm",
                (loading || notAuth || !toolsReady) && "opacity-50 cursor-not-allowed"
              )}
            >
              <RotateCcw className="w-4 h-4" /> Restore...
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
