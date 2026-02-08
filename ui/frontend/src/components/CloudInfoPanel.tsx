import { Cloud, User, Hash, MapPin, Database } from 'lucide-react'
import type { AuthStatus } from '../types'

interface CloudInfoPanelProps {
  auth: AuthStatus | null
  onTFClick: () => void
}

export function CloudInfoPanel({ auth, onTFClick }: CloudInfoPanelProps) {
  if (!auth) return null

  const infoItems = [
    { icon: Cloud, label: 'Provider', value: auth.provider.toUpperCase() },
    { icon: User, label: 'Account', value: auth.account || 'Not Authenticated' },
    { icon: Hash, label: 'Project ID', value: auth.project || 'None' },
    { icon: MapPin, label: 'Region', value: auth.region || 'None' },
  ]

  return (
    <div className="w-64 border-r bg-card flex flex-col">
      <div className="p-4 border-b">
        <h2 className="font-semibold text-sm uppercase tracking-wider text-muted-foreground">
          Cloud Environment
        </h2>
      </div>
      <div className="flex-1 overflow-auto p-4 space-y-6">
        {infoItems.map((item) => (
          <div key={item.label} className="space-y-1">
            <div className="flex items-center gap-2 text-muted-foreground">
              <item.icon className="w-4 h-4" />
              <span className="text-xs font-medium">{item.label}</span>
            </div>
            <div className="text-sm font-medium truncate" title={item.value}>
              {item.value}
            </div>
          </div>
        ))}
        
        {/* Special case for TF State with click handler */}
        <div className="space-y-1">
          <div className="flex items-center gap-2 text-muted-foreground">
            <Database className="w-4 h-4" />
            <span className="text-xs font-medium">TF State</span>
          </div>
          <button 
            onClick={onTFClick}
            className="text-sm font-medium truncate text-blue-500 hover:underline text-left"
          >
            View Resources
          </button>
        </div>
      </div>
    </div>
  )
}