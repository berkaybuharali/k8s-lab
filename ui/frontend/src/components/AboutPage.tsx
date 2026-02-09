import { ArrowLeft } from 'lucide-react'
import kubernetesLogo from '@/assets/kubernetes_logo.svg'
import talosLogo from '@/assets/talos_logo.svg'
import veleroLogo from '@/assets/velero_logo.svg'
import terraformLogo from '@/assets/terraform_logo.svg'
import nginxLogo from '@/assets/nginx_logo.svg'
import redisLogo from '@/assets/redis_logo.svg'

interface AboutPageProps {
  onBack: () => void
}

export function AboutPage({ onBack }: AboutPageProps) {
  const stack = [
    { name: 'Terraform', desc: 'Infrastructure as Code for cloud provisioning', logo: terraformLogo },
    { name: 'Talos Linux', desc: 'Minimal, immutable OS purpose-built for Kubernetes', logo: talosLogo },
    { name: 'Kubernetes', desc: 'Container orchestration on bare-metal VMs', logo: kubernetesLogo },
    { name: 'Velero', desc: 'Backup and restore for cluster resources and volumes', logo: veleroLogo },
    { name: 'NGINX', desc: 'Web server workload with persistent storage', logo: nginxLogo },
    { name: 'Redis', desc: 'In-memory data store for testing backup/restore cycles', logo: redisLogo },
  ]

  return (
    <div className="space-y-6 max-w-3xl mx-auto">
      <button onClick={onBack} className="flex items-center gap-2 text-sm text-muted-foreground hover:text-foreground transition-colors">
        <ArrowLeft className="w-4 h-4" /> Back to Dashboard
      </button>

      <div className="border rounded-xl bg-card shadow-sm overflow-hidden">
        <div className="p-8 space-y-6">
          <div>
            <h1 className="text-3xl font-bold">k8s<span className="text-primary">-lab</span></h1>
            <p className="text-muted-foreground mt-2">
              A reproducible Kubernetes lab environment on cloud VMs with full lifecycle management.
            </p>
          </div>

          <div className="space-y-3">
            <h2 className="text-lg font-semibold">What is this?</h2>
            <p className="text-sm text-muted-foreground leading-relaxed">
              k8s-lab provisions a complete Kubernetes cluster on cloud VMs (currently GCP) using Terraform and Talos Linux.
              It is designed for learning, experimentation, and proof-of-concept work with a focus on cost-efficiency
              through daily create/destroy cycles. The entire stack can be deployed and torn down in minutes.
            </p>
          </div>

          <div className="space-y-3">
            <h2 className="text-lg font-semibold">Architecture</h2>
            <p className="text-sm text-muted-foreground leading-relaxed">
              The lab uses a layered architecture: Infrastructure (VMs, networking) at the bottom,
              followed by the Kubernetes cluster (Talos Linux), platform tools (Velero for backup/restore, CSI driver),
              and user applications (NGINX, Redis) at the top. Layers 3-4 are cloud-agnostic and reusable across providers.
            </p>
          </div>

          <div className="space-y-3">
            <h2 className="text-lg font-semibold">Technology Stack</h2>
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
              {stack.map(item => (
                <div key={item.name} className="flex items-start gap-3 p-3 rounded-lg bg-muted/40 border">
                  <img src={item.logo} alt={item.name} className="w-6 h-6 flex-shrink-0 mt-0.5" />
                  <div>
                    <div className="text-sm font-semibold">{item.name}</div>
                    <div className="text-xs text-muted-foreground">{item.desc}</div>
                  </div>
                </div>
              ))}
            </div>
          </div>

          <div className="space-y-3">
            <h2 className="text-lg font-semibold">Daily Lifecycle</h2>
            <p className="text-sm text-muted-foreground leading-relaxed">
              To minimize cloud costs, the lab supports a daily create/destroy workflow. Deploy in the morning,
              work throughout the day, backup your state, and destroy at night. Next day, restore from backup
              and continue where you left off. All operations are available through this dashboard or via CLI.
            </p>
          </div>

          <div className="pt-4 border-t text-xs text-muted-foreground">
            <a href="https://github.com/berkaybuharali/k8s-lab" target="_blank" rel="noopener noreferrer" className="text-primary hover:underline">
              github.com/berkaybuharali/k8s-lab
            </a>
          </div>
        </div>
      </div>
    </div>
  )
}
