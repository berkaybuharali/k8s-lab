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

/**
 * Layout wraps every page with the persistent header/navigation bar.
 * Receives view state and setView so nav buttons can switch pages.
 * Extracted from AppInner (Item 5.12).
 */
export function Layout({ children, view, setView, auth, status, isRunning }: LayoutProps) {
  return (
    <div className="h-screen bg-background text-foreground flex flex-col overflow-hidden">
      <header className="h-14 border-b px-4 flex items-center justify-between bg-card shrink-0 sticky top-0 z-10">
        <div className="flex items-center gap-4">
          <div className="flex items-center gap-2">
            <button onClick={() => setView('dashboard')} className="font-bold text-lg tracking-tight hover:opacity-80 transition-opacity">
              k8s<span className="text-primary">-lab</span>
            </button>
            <div className="h-4 w-px bg-border mx-2" />
            <button onClick={() => setView('dashboard')} className={cn("text-sm transition-colors hover:text-foreground", view === 'dashboard' ? "text-foreground font-medium" : "text-muted-foreground")}>
              Infrastructure
            </button>
            <button onClick={() => setView('architecture')} className={cn("text-sm transition-colors hover:text-foreground", view === 'architecture' ? "text-foreground font-medium" : "text-muted-foreground")}>
              Architecture
            </button>
            <button onClick={() => setView('shop')} className={cn("text-sm transition-colors hover:text-foreground", view === 'shop' ? "text-foreground font-medium" : "text-muted-foreground")}>
              Magic Cake Shop
            </button>
            <button onClick={() => setView('backoffice')} className={cn("text-sm transition-colors hover:text-foreground", view === 'backoffice' ? "text-foreground font-medium" : "text-muted-foreground")}>
              Backoffice
            </button>
            <button onClick={() => setView('about')} className={cn("text-sm transition-colors hover:text-foreground", view === 'about' ? "text-foreground font-medium" : "text-muted-foreground")}>
              About
            </button>
          </div>

          {isRunning && (
            <div className="flex items-center gap-2 text-sm text-muted-foreground border-l pl-4">
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
