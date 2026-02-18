import { BackofficeMap } from './BackofficeMap'
import { BackofficeOrders } from './BackofficeOrders'
import { BackofficeInventory } from './BackofficeInventory'
import { BackofficeRevenue } from './BackofficeRevenue'
import { BackofficeLog } from './BackofficeLog'
import { AgentChat } from './AgentChat'

export function BackofficePage() {
  return (
    <div className="h-full overflow-y-auto bg-muted/10 p-6 space-y-6">
      
      <div className="flex justify-between items-center">
        <h1 className="text-2xl font-bold tracking-tight">Backoffice</h1>
        <div className="text-sm text-muted-foreground">
            {new Date().toLocaleDateString(undefined, { weekday: 'long', year: 'numeric', month: 'long', day: 'numeric' })}
        </div>
      </div>

      <BackofficeRevenue />

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          <div className="lg:col-span-2 space-y-6">
              <BackofficeMap />
              <BackofficeOrders />
          </div>
          
          <div className="space-y-6">
              <BackofficeInventory />
              
              <div className="border rounded-xl bg-card shadow-sm p-1 overflow-hidden">
                <div className="px-3 py-2 text-xs font-semibold text-muted-foreground uppercase tracking-wider border-b mb-1">
                    Supply Chain Agent
                </div>
                <AgentChat system="supply-chain" className="h-[300px] border-none shadow-none" placeholder="Ask about routes, stock, or orders..." />
              </div>

              <div className="h-[250px]">
                  <BackofficeLog />
              </div>
          </div>
      </div>
    </div>
  )
}
