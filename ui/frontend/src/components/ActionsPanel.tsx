import { Trash, Layers, Database, Server, AppWindow } from 'lucide-react'
import type { GlobalStatus } from '../types'
import { cn } from '@/lib/utils'

interface ActionsPanelProps {
  status: GlobalStatus | null
  onTrigger: (op: string) => void
  loading: boolean
}

export function ActionsPanel({ status, onTrigger, loading }: ActionsPanelProps) {
  const k8sReady = status?.k8s === 'Ready'
  const toolsReady = status?.tools === 'Installed'

  const actions = [
    {
      id: 'deploy-infra',
      label: 'Deploy Infra',
      icon: Server,
      disabled: loading,
      desc: "Terraform + Talos Bootstrap"
    },
    {
      id: 'deploy-tools',
      label: 'Deploy Tools',
      icon: Layers,
      disabled: loading || !k8sReady, // Need K8s API
      desc: "CSI Driver, Velero"
    },
    {
      id: 'deploy-applications',
      label: 'Deploy Apps',
      icon: AppWindow,
      disabled: loading || !toolsReady, // Need StorageClass
      desc: "NGINX, Redis"
    },
    {
      id: 'seed-redis',
      label: 'Seed Data',
      icon: Database,
      disabled: loading || status?.apps !== 'Deployed',
      desc: "Fill Redis with test data"
    }
  ]

  return (
    <div className="p-6 border rounded-lg shadow-sm bg-card text-card-foreground">
      <h2 className="font-semibold mb-4">Quick Actions</h2>
      <div className="space-y-3">
        {actions.map((action) => (
          <button
            key={action.id}
            onClick={() => onTrigger(action.id)}
            disabled={action.disabled}
            className={cn(
              "w-full flex items-center gap-3 p-3 rounded-md border transition-all text-left",
              action.disabled 
                ? "opacity-50 cursor-not-allowed bg-muted" 
                : "hover:bg-accent hover:text-accent-foreground"
            )}
          >
            <div className={cn("p-2 rounded-full bg-primary/10 text-primary")}>
              <action.icon className="w-4 h-4" />
            </div>
            <div>
              <div className="font-medium text-sm">{action.label}</div>
              <div className="text-xs text-muted-foreground">{action.desc}</div>
            </div>
          </button>
        ))}

        <div className="border-t my-4" />

        <button
          onClick={() => onTrigger('destroy')}
          disabled={loading}
          className={cn(
            "w-full flex items-center gap-3 p-3 rounded-md border border-destructive/20 transition-all text-left hover:bg-destructive/10 text-destructive",
            loading && "opacity-50 cursor-not-allowed"
          )}
        >
          <div className="p-2 rounded-full bg-destructive/10">
            <Trash className="w-4 h-4" />
          </div>
          <div>
            <div className="font-medium text-sm">Destroy All</div>
            <div className="text-xs opacity-80">Teardown infrastructure</div>
          </div>
        </button>
      </div>
    </div>
  )
}