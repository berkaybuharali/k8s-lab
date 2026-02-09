import { Package, Loader2 } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { GlobalStatus } from '../types'
import gceLogo from '@/assets/gce_logo.svg'
import kubernetesLogo from '@/assets/kubernetes_logo.svg'
import talosLogo from '@/assets/talos_logo.svg'
import veleroLogo from '@/assets/velero_logo.svg'
import nginxLogo from '@/assets/nginx_logo.svg'
import redisLogo from '@/assets/redis_logo.svg'

interface StatusPanelProps {
  status: GlobalStatus | null
  isRunning?: boolean
  runningOp?: string
  provider?: string
}

export function StatusPanel({ status, isRunning, runningOp, provider }: StatusPanelProps) {
  const deployingLayer = isRunning ? (
    runningOp?.includes('deploy-infra') || runningOp === 'deploy' ? 'infra' :
    runningOp?.includes('deploy-tools') ? 'tools' :
    runningOp?.includes('deploy-applications') ? 'apps' :
    runningOp === 'destroy' ? 'infra' : null
  ) : null

  const cards = [
    {
      key: 'infra',
      label: 'Infrastructure',
      subtitle: provider === 'gcp' ? 'GCE VMs, VPC, Firewall' : 'VMs, Network',
      value: deployingLayer === 'infra' ? 'Deploying...' : (status?.infra || 'Unknown'),
      logo: provider === 'gcp' ? gceLogo : undefined,
      iconBg: 'bg-blue-500/15',
      ok: status?.infra === 'Running',
      deploying: deployingLayer === 'infra',
    },
    {
      key: 'k8s',
      label: 'Kubernetes',
      subtitle: 'Talos Linux Cluster',
      value: deployingLayer === 'infra' ? 'Waiting...' : (status?.k8s || 'Unknown'),
      logos: [kubernetesLogo, talosLogo],
      iconBg: 'bg-purple-500/15',
      ok: status?.k8s === 'Ready',
      deploying: false,
    },
    {
      key: 'tools',
      label: 'Platform Tools',
      subtitle: 'Velero, CSI Driver',
      value: deployingLayer === 'tools' ? 'Installing...' : (deployingLayer === 'infra' ? 'Waiting...' : (status?.tools || 'Unknown')),
      logo: veleroLogo,
      iconBg: 'bg-cyan-500/15',
      ok: status?.tools === 'Installed',
      deploying: deployingLayer === 'tools',
    },
    {
      key: 'apps',
      label: 'Applications',
      subtitle: 'NGINX, Redis',
      value: deployingLayer === 'apps' ? 'Deploying...' : (deployingLayer ? 'Waiting...' : (status?.apps || 'Unknown')),
      logos: [nginxLogo, redisLogo],
      iconBg: 'bg-green-500/15',
      ok: status?.apps === 'Deployed',
      deploying: deployingLayer === 'apps',
    },
  ]

  return (
    <div className="border rounded-xl bg-card shadow-sm overflow-hidden">
      <div className="px-5 py-3 border-b">
        <h2 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">Cluster Status</h2>
      </div>
      <div className="p-5">
        <div className="grid grid-cols-2 gap-3">
          {cards.map((card) => (
            <div key={card.key} className={cn(
              "flex items-center gap-3 p-4 rounded-lg bg-muted/40 border",
              card.deploying && "border-primary/40"
            )}>
              <div className={cn("w-10 h-10 rounded-lg flex items-center justify-center gap-1 flex-shrink-0", card.iconBg)}>
                {'logos' in card && card.logos ? (
                  card.logos.map((l, i) => <img key={i} src={l} alt="" className="w-4 h-4" />)
                ) : card.logo ? (
                  <img src={card.logo} alt={card.label} className="w-5 h-5" />
                ) : (
                  <Package className="w-5 h-5 text-blue-500" />
                )}
              </div>
              <div>
                <div className="text-sm font-semibold">{card.label}</div>
                <div className="text-[10px] text-muted-foreground mb-1">{card.subtitle}</div>
                <div className="text-sm font-semibold flex items-center gap-1.5">
                  {card.deploying ? (
                    <Loader2 className="w-3 h-3 animate-spin text-primary" />
                  ) : (
                    <div className={cn("w-2 h-2 rounded-full", card.ok ? "bg-green-500" : "bg-muted-foreground")} />
                  )}
                  {card.value}
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
