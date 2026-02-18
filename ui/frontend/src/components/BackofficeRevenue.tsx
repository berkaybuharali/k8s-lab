import { useEffect, useState } from 'react'
import { DollarSign, ShoppingBag, TrendingUp } from 'lucide-react'

export function BackofficeRevenue() {
  const [stats, setStats] = useState({ count: 0, revenue: 0, average: 0 })

  useEffect(() => {
    fetch('/api/orders/stats')
      .then(res => res.json())
      .then(setStats)
      .catch(console.error)
  }, [])

  const cards = [
    { label: "Total Revenue", value: `€${stats.revenue.toFixed(2)}`, icon: DollarSign, color: "text-green-500" },
    { label: "Total Orders", value: stats.count.toString(), icon: ShoppingBag, color: "text-blue-500" },
    { label: "Avg. Order", value: `€${stats.average.toFixed(2)}`, icon: TrendingUp, color: "text-purple-500" },
  ]

  return (
    <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
      {cards.map((card, i) => (
        <div key={i} className="bg-card border rounded-xl p-4 shadow-sm flex items-center gap-4">
            <div className={`p-3 rounded-full bg-muted/50 ${card.color}`}>
                <card.icon className="w-5 h-5" />
            </div>
            <div>
                <div className="text-xs text-muted-foreground font-medium uppercase tracking-wide">{card.label}</div>
                <div className="text-xl font-bold">{card.value}</div>
            </div>
        </div>
      ))}
    </div>
  )
}
