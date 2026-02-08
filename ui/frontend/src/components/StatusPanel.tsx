import { Server, Activity, Package, AppWindow } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { GlobalStatus } from '../types'

export function StatusPanel({ status }: { status: GlobalStatus | null }) {
  const cards = [
    {
      label: 'Infrastructure',
      value: status?.infra || 'Unknown',
      icon: Server,
      color: status?.infra === 'Running' ? 'text-green-500' : 'text-red-500',
    },
    {
      label: 'Kubernetes',
      value: status?.k8s || 'Unknown',
      icon: Activity,
      color: status?.k8s === 'Ready' ? 'text-green-500' : 'text-red-500',
    },
    {
      label: 'Platform Tools',
      value: status?.tools || 'Unknown',
      icon: Package,
      color: status?.tools === 'Installed' ? 'text-green-500' : 'text-yellow-500',
    },
    {
      label: 'Applications',
      value: status?.apps || 'Unknown',
      icon: AppWindow,
      color: status?.apps === 'Deployed' ? 'text-green-500' : 'text-yellow-500',
    },
  ]

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
      {cards.map((card) => (
        <div key={card.label} className="p-4 border rounded-lg bg-card shadow-sm">
          <div className="flex items-center justify-between mb-2">
            <span className="text-sm font-medium text-muted-foreground">{card.label}</span>
            <card.icon className={cn("w-4 h-4", card.color)} />
          </div>
          <div className="text-xl font-bold">{card.value}</div>
        </div>
      ))}
    </div>
  )
}
