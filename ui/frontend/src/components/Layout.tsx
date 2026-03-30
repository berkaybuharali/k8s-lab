import { cn } from "@/lib/utils"
import { Loader2 } from 'lucide-react'
import { ThemeToggle } from './ThemeToggle'
import type { AuthStatus, GlobalStatus } from '../types'

type ViewName = 'dashboard' | 'pod-detail' | 'tf-detail' | 'about' | 'architecture' | 'shop' | 'backoffice'

interface LayoutProps {
  children: React.ReactNode
  view: ViewName
  setView: (v: ViewName) => void
  auth: AuthStatus | null
  status: GlobalStatus | null
  isRunning: boolean
}

export function Layout({ children, view, setView, auth, status, isRunning }: LayoutProps) {
  const navItem = (v: ViewName, label: string) => (
    <button
      onClick={() => setView(v)}
      className={cn(
        "text-sm px-3 py-1.5 rounded-md border transition-colors",
        view === v
          ? "bg-accent border-border text-foreground font-medium"
          : "border-border/40 text-muted-foreground hover:bg-accent hover:text-foreground hover:border-border"
      )}
    >
      {label}
    </button>
  )

  return (
    <div className="h-screen bg-background text-foreground flex flex-col overflow-hidden">
      <header className="h-14 border-b px-4 flex items-center justify-between bg-card shrink-0 sticky top-0 z-10 shadow-sm">
        <div className="flex items-center gap-1">
          {/* Brand */}
          <button
            onClick={() => setView('dashboard')}
            className="font-bold text-lg tracking-tight hover:opacity-80 transition-opacity mr-2"
          >
            k8s<span className="text-primary">-lab</span>
          </button>

          {/* Platform group */}
          <div className="h-4 w-px bg-border mx-1" />
          {navItem('dashboard', 'Infrastructure')}
          {navItem('architecture', 'Architecture')}

          {/* Shop group */}
          <div className="h-4 w-px bg-border mx-1" />
          {navItem('shop', 'Magic Cake Shop')}
          {navItem('backoffice', 'Backoffice')}

          {/* Info group */}
          <div className="h-4 w-px bg-border mx-1" />
          {navItem('about', 'About')}

          {isRunning && (
            <div className="flex items-center gap-2 text-sm text-muted-foreground border-l pl-4 ml-2">
              <Loader2 className="w-4 h-4 animate-spin text-primary" />
              <span>Operation in progress...</span>
            </div>
          )}
        </div>

        <div className="flex items-center gap-4">
          {status && (
            <div className={cn(
              "flex items-center gap-2 text-xs px-3 py-1 rounded-full border font-medium",
              status.tunnel === 'Connected' ? "bg-green-500/10 border-green-500/20 text-green-600" :
              status.tunnel === 'Reconnecting' ? "bg-yellow-500/10 border-yellow-500/20 text-yellow-600 animate-pulse" :
              status.tunnel === 'Starting' ? "bg-yellow-500/10 border-yellow-500/20 text-yellow-600" :
              (status.tunnel === 'Idle' || status.tunnel === 'N/A') ? "border-border text-muted-foreground" :
              "bg-red-500/10 border-red-500/20 text-red-600"
            )}>
              <div className={cn("w-2 h-2 rounded-full",
                status.tunnel === 'Connected' ? "bg-green-500" :
                status.tunnel === 'Reconnecting' || status.tunnel === 'Starting' ? "bg-yellow-500" :
                (status.tunnel === 'Idle' || status.tunnel === 'N/A') ? "bg-muted-foreground" :
                "bg-red-500"
              )} />
              <span>Tunnel: {status.tunnel}</span>
            </div>
          )}

          {auth && (
            <div className={cn(
              "flex items-center gap-2 text-xs px-3 py-1 rounded-full border",
              auth.authenticated ? "bg-green-500/10 border-green-500/20 text-green-600" : "bg-red-500/10 border-red-500/20 text-red-600"
            )}>
              <div className={cn("w-2 h-2 rounded-full", auth.authenticated ? "bg-green-500" : "bg-red-500")} />
              <span>{auth.authenticated ? "Authenticated" : "Not Authenticated"}</span>
            </div>
          )}

          <ThemeToggle />
        </div>
      </header>

      {children}
    </div>
  )
}
