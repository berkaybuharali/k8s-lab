import { useEffect, useState } from 'react'
import { cn } from '@/lib/utils'
import { Package, Calendar, MapPin, ChevronDown, ChevronUp, Trash2 } from 'lucide-react'

interface Order {
  order_id: string
  customer_name: string
  address: string
  delivery_date: string
  total_price: string
  status: string
  cakes: string // JSON string
  image_paths: string // JSON string
}

export function BackofficeOrders() {
  const [orders, setOrders] = useState<Order[]>([])
  const [expandedId, setExpandedId] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    fetch('/api/orders')
      .then(res => res.json())
      .then(data => {
          // Sort by date desc
          const sorted = Array.isArray(data) ? data.sort((a: any, b: any) => 
            new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
          ) : []
          setOrders(sorted)
      })
      .catch(console.error)
      .finally(() => setLoading(false))
  }, [])

  const toggleExpand = (id: string) => {
    setExpandedId(expandedId === id ? null : id)
  }

  if (loading) return <div className="h-64 flex items-center justify-center text-muted-foreground">Loading orders...</div>

  return (
    <div className="border rounded-xl bg-card shadow-sm overflow-hidden">
      <div className="px-5 py-3 border-b flex justify-between items-center">
        <h2 className="text-sm font-semibold">Active Orders</h2>
        <span className="text-xs text-muted-foreground">{orders.length} orders</span>
      </div>
      
      <div className="divide-y">
        {orders.length === 0 ? (
            <div className="p-8 text-center text-muted-foreground text-sm">No orders found. Seed data to get started.</div>
        ) : (
            orders.map(order => {
                let cakes: any[] = []
                let images: string[] = []
                try { cakes = JSON.parse(order.cakes || '[]') } catch { /* malformed JSON — show empty */ }
                try { images = JSON.parse(order.image_paths || '[]') } catch { /* malformed JSON — show empty */ }
                const isExpanded = expandedId === order.order_id

                return (
                    <div key={order.order_id} className="text-sm group">
                        <div 
                            className={cn("flex items-center gap-4 p-4 cursor-pointer hover:bg-muted/50 transition-colors", isExpanded && "bg-muted/30")}
                            onClick={() => toggleExpand(order.order_id)}
                        >
                            <div className="w-8 h-8 rounded-full bg-primary/10 flex items-center justify-center shrink-0">
                                <Package className="w-4 h-4 text-primary" />
                            </div>
                            
                            <div className="flex-1 min-w-0 grid grid-cols-2 md:grid-cols-4 gap-4">
                                <div>
                                    <div className="font-medium truncate">{order.customer_name}</div>
                                    <div className="text-xs text-muted-foreground font-mono">{order.order_id}</div>
                                </div>
                                <div className="hidden md:block">
                                    <div className="flex items-center gap-1 text-muted-foreground">
                                        <MapPin className="w-3 h-3" />
                                        <span className="truncate">{order.address}</span>
                                    </div>
                                </div>
                                <div className="hidden md:block">
                                    <div className="flex items-center gap-1 text-muted-foreground">
                                        <Calendar className="w-3 h-3" />
                                        <span>{order.delivery_date}</span>
                                    </div>
                                </div>
                                <div className="text-right font-medium">
                                    €{parseFloat(order.total_price).toFixed(2)}
                                </div>
                            </div>

                            <div className="text-muted-foreground">
                                {isExpanded ? <ChevronUp className="w-4 h-4" /> : <ChevronDown className="w-4 h-4" />}
                            </div>
                        </div>

                        {/* Details View */}
                        {isExpanded && (
                            <div className="bg-muted/30 p-4 border-t border-b grid grid-cols-1 md:grid-cols-2 gap-6 animate-in slide-in-from-top-2 duration-200">
                                <div className="space-y-4">
                                    <h4 className="font-semibold text-xs uppercase tracking-wider text-muted-foreground">Cake Details</h4>
                                    {cakes.map((cake: any, i: number) => (
                                        <div key={i} className="bg-background border rounded-lg p-3 text-xs space-y-1">
                                            <div className="font-medium text-sm flex justify-between">
                                                <span>Cake #{i+1}</span>
                                                <span className="text-muted-foreground">{cake.people_count} people</span>
                                            </div>
                                            <div className="grid grid-cols-2 gap-2 text-muted-foreground">
                                                <span>Flavor: <span className="text-foreground">{cake.flavor}</span></span>
                                                <span>Nuts: <span className="text-foreground">{cake.nuts}</span></span>
                                                <span className="col-span-2">Concept: <span className="text-foreground">{cake.concept}</span></span>
                                            </div>
                                        </div>
                                    ))}
                                    
                                    <div className="pt-2 flex justify-end">
                                        <button className="text-destructive hover:bg-destructive/10 px-3 py-1.5 rounded-md text-xs font-medium flex items-center gap-2 transition-colors">
                                            <Trash2 className="w-3 h-3" /> Delete Order
                                        </button>
                                    </div>
                                </div>

                                <div className="space-y-4">
                                    <h4 className="font-semibold text-xs uppercase tracking-wider text-muted-foreground">Generated Designs</h4>
                                    <div className="grid grid-cols-2 gap-2">
                                        {images.map((_path: string, i: number) => (
                                            <div key={i} className="aspect-square bg-muted rounded-lg overflow-hidden border relative group/img">
                                                {/* In a real app we'd sign this GCS URL. For now we use a placeholder or check if we can proxy it. 
                                                    Since we can't easily proxy GCS without signed URLs, we'll show a placeholder if it's a gs:// path 
                                                    or try to load it if it's http.
                                                */}
                                                <div className="absolute inset-0 flex items-center justify-center text-xs text-muted-foreground p-2 text-center bg-background/50">
                                                    Image preview requires signed URL<br/>(See backend logs)
                                                </div>
                                                {/* <img src={path} className="w-full h-full object-cover" /> */}
                                            </div>
                                        ))}
                                    </div>
                                </div>
                            </div>
                        )}
                    </div>
                )
            })
        )}
      </div>
    </div>
  )
}
