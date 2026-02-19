import { Info, WifiOff, XCircle } from 'lucide-react'
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

  // Priority 2: Tunnel Lost (infra exists but tunnel dropped)
  if (status && status.infra === 'Running' && status.tunnel !== 'Connected' && status.tunnel !== 'Starting' && status.tunnel !== 'Idle') {
    return (
      <div className="bg-yellow-500/15 border-b border-yellow-500/20 px-4 py-2 flex items-center justify-center gap-2 text-yellow-600 dark:text-yellow-500 text-sm font-medium animate-pulse">
        <WifiOff className="w-4 h-4" />
        <span>Tunnel disconnected. Reconnecting... Cluster data may be stale.</span>
      </div>
    )
  }

  // Priority 3: No infrastructure yet — guide the user to get started
  if (status && status.infra === 'Not Created') {
    return (
      <div className="bg-primary/8 border-b border-primary/15 px-4 py-2 flex items-center justify-center gap-2 text-primary/80 text-sm">
        <Info className="w-4 h-4 shrink-0" />
        <span>No cluster found. Use <strong>Deploy All</strong> for a one-click setup or follow the step-by-step buttons in the Actions panel.</span>
      </div>
    )
  }

  return null
}
