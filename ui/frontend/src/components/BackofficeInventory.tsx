import { useEffect, useState } from 'react'
import { cn } from '@/lib/utils'

export function BackofficeInventory() {
  const [inventory, setInventory] = useState<Record<string, number>>({})
  const [loading, setLoading] = useState(true)

  const fetchInventory = () => {
    fetch('/api/inventory')
      .then(res => res.json())
      .then(setInventory)
      .catch(console.error)
      .finally(() => setLoading(false))
  }

  useEffect(() => {
    fetchInventory()
    const interval = setInterval(fetchInventory, 10000)
    return () => clearInterval(interval)
  }, [])

  const MAX_STOCK = 5
  
  const getStatusColor = (qty: number) => {
      if (qty <= 1) return "bg-red-500"
      if (qty <= 2) return "bg-yellow-500"
      return "bg-green-500"
  }

  return (
    <div className="border rounded-xl bg-card shadow-sm p-5 space-y-4">
      <h3 className="font-semibold text-sm">Live Inventory</h3>
      
      {loading ? (
          <div className="space-y-3 animate-pulse">
              {[1,2,3,4,5].map(i => <div key={i} className="h-8 bg-muted rounded-md" />)}
          </div>
      ) : (
          <div className="space-y-4">
              {Object.entries(inventory).map(([item, qty]) => (
                  <div key={item} className="space-y-1.5">
                      <div className="flex justify-between text-xs font-medium">
                          <span className="capitalize">{item}</span>
                          <span className={cn(
                              qty === 0 ? "text-destructive" : "text-muted-foreground"
                          )}>{qty} / {MAX_STOCK}</span>
                      </div>
                      <div className="h-2 w-full bg-muted rounded-full overflow-hidden">
                          <div 
                            className={cn("h-full transition-all duration-500", getStatusColor(qty))} 
                            style={{ width: `${(Math.max(0, Math.min(qty, MAX_STOCK)) / MAX_STOCK) * 100}%` }} 
                          />
                      </div>
                  </div>
              ))}
          </div>
      )}
    </div>
  )
}
