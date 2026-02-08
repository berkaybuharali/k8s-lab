import { useEffect, useState } from 'react'
import { ArrowLeft, Box, CheckCircle } from 'lucide-react'

interface TerraformResource {
  address: string
  type: string
  name: string
  values: any
}

interface TerraformState {
  values?: {
    root_module?: {
      resources?: TerraformResource[]
    }
  }
}

export function TerraformResources({ onBack }: { onBack: () => void }) {
  const [resources, setResources] = useState<TerraformResource[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    fetch('/api/terraform/resources')
      .then(res => res.json())
      .then((data: TerraformState) => {
        setResources(data.values?.root_module?.resources || [])
      })
      .catch(console.error)
      .finally(() => setLoading(false))
  }, [])

  if (loading) return <div className="p-8 text-center">Loading resources...</div>

  // Group by meaningful categories
  const grouped: Record<string, TerraformResource[]> = {}
  resources.forEach(r => {
    let category = 'Other'
    if (r.type.includes('compute_instance') || r.type.includes('compute_disk')) category = 'Compute'
    else if (r.type.includes('network') || r.type.includes('firewall') || r.type.includes('subnetwork') || r.type.includes('address')) category = 'Network'
    else if (r.type.includes('iam') || r.type.includes('service_account') || r.type.includes('project_iam')) category = 'IAM'
    else if (r.type.includes('storage') || r.type.includes('bucket')) category = 'Storage'
    
    if (!grouped[category]) grouped[category] = []
    grouped[category].push(r)
  })

  // Sort categories for consistent order
  const categories = ['Network', 'Compute', 'IAM', 'Storage', 'Other'].filter(c => grouped[c])

  return (
    <div className="space-y-4">
      <button onClick={onBack} className="flex items-center gap-2 text-sm text-muted-foreground hover:text-foreground transition-colors">
        <ArrowLeft className="w-4 h-4" /> Back to Dashboard
      </button>

      <div className="border rounded-lg bg-card shadow-sm overflow-hidden">
        <div className="p-6 border-b flex justify-between items-center">
          <h1 className="text-2xl font-bold flex items-center gap-3">
            <Box className="w-6 h-6 text-primary" /> Terraform Resources
          </h1>
          <div className="text-sm text-muted-foreground">{resources.length} resources managed</div>
        </div>

        <div className="p-6 space-y-6">
          {resources.length === 0 && <div className="text-center text-muted-foreground">No resources found.</div>}
          
          {categories.map((category) => (
            <div key={category}>
              <h3 className="font-semibold text-sm uppercase tracking-wider text-muted-foreground mb-3 border-b pb-1">{category}</h3>
              <div className="space-y-2">
                {grouped[category].map(r => (
                  <div key={r.address} className="flex items-center justify-between p-3 border rounded-md hover:bg-muted/50 transition-colors bg-background">
                    <div className="flex flex-col">
                      <span className="font-mono text-sm font-medium">{r.name}</span>
                      <span className="text-xs text-muted-foreground">{r.type}</span>
                    </div>
                    <div className="flex items-center gap-2 text-green-600 text-xs font-medium">
                      <CheckCircle className="w-4 h-4" /> Managed
                    </div>
                  </div>
                ))}
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
