import { WifiOff, XCircle } from 'lucide-react'
import type { AuthStatus, GlobalStatus } from '../types'

interface BannerProps {
  auth: AuthStatus | null
  status: GlobalStatus | null
}

export function Banner({ auth, status }: BannerProps) {
  // Priority 1: Auth Failure
  if (auth && !auth.authenticated) {
    return (
      <div className="bg-destructive/15 border-b border-destructive/20 px-4 py-2 flex items-center justify-center gap-2 text-destructive text-sm font-medium">
        <XCircle className="w-4 h-4" />
        <span>Authentication failed. Run <code className="bg-destructive/15 px-1.5 py-0.5 rounded text-xs font-mono">gcloud auth application-default login</code> and restart.</span>
      </div>
    )
  }

  // Priority 2: Tunnel Lost (and was previously connected/running, assumed if infra exists)
  if (status && status.infra === 'Running' && status.tunnel !== 'Connected' && status.tunnel !== 'Starting' && status.tunnel !== 'Idle') {
    return (
      <div className="bg-yellow-500/15 border-b border-yellow-500/20 px-4 py-2 flex items-center justify-center gap-2 text-yellow-600 dark:text-yellow-500 text-sm font-medium animate-pulse">
        <WifiOff className="w-4 h-4" />
        <span>Tunnel disconnected. Reconnecting... Cluster data may be stale.</span>
      </div>
    )
  }

  return null
}
