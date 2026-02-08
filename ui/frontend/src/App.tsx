import { useEffect, useState } from 'react'
import { cn } from "@/lib/utils"

interface AuthStatus {
  authenticated: boolean
  account?: string
  project?: string
  region?: string
  provider: string
  error?: string
}

function App() {
  const [auth, setAuth] = useState<AuthStatus | null>(null)

  useEffect(() => {
    fetch('/api/auth')
      .then(res => res.json())
      .then(data => setAuth(data))
      .catch(err => console.error("Failed to fetch auth:", err))
  }, [])

  return (
    <div className="min-h-screen bg-background text-foreground flex flex-col">
       <header className="h-14 border-b px-4 flex items-center justify-between bg-card">
         <div className="flex items-center gap-2">
            <div className="w-8 h-8 bg-primary rounded flex items-center justify-center text-primary-foreground font-bold">
              K
            </div>
            <h1 className="font-semibold text-lg">Kubernetes Lab</h1>
         </div>
         
         <div className="flex items-center gap-4">
             {auth && (
                 <>
                    <div className="flex items-center gap-2 text-sm">
                        <span className="text-muted-foreground">Cloud:</span>
                        <span className="font-medium uppercase">{auth.provider}</span>
                    </div>
                    
                    <div className={cn(
                        "flex items-center gap-2 text-sm px-3 py-1 rounded-full border",
                        auth.authenticated ? "bg-green-500/10 border-green-500/20 text-green-600 dark:text-green-400" : "bg-red-500/10 border-red-500/20 text-red-600 dark:text-red-400"
                    )}>
                        <div className={cn("w-2 h-2 rounded-full", auth.authenticated ? "bg-green-500" : "bg-red-500")} />
                        <span>{auth.authenticated ? (auth.account || "Authenticated") : "Not Authenticated"}</span>
                    </div>
                 </>
             )}
         </div>
       </header>
       <main className="flex-1 p-6 grid gap-6 grid-cols-1 md:grid-cols-2 lg:grid-cols-3">
          <div className="p-6 border rounded-lg shadow-sm bg-card text-card-foreground">
            <h2 className="font-semibold mb-2">Foundation</h2>
            <p className="text-sm text-muted-foreground">Frontend scaffolded successfully.</p>
          </div>
          
          {auth && auth.project && (
              <div className="p-6 border rounded-lg shadow-sm bg-card text-card-foreground">
                <h2 className="font-semibold mb-2">Project Info</h2>
                <div className="text-sm space-y-1">
                    <p><span className="text-muted-foreground">Project:</span> {auth.project}</p>
                    <p><span className="text-muted-foreground">Region:</span> {auth.region}</p>
                </div>
              </div>
          )}
       </main>
    </div>
  )
}

export default App
