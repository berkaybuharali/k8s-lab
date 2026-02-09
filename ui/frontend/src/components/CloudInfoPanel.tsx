import { User, Hash, MapPin } from 'lucide-react'
import type { AuthStatus } from '../types'
import gcpLogo from '@/assets/gcp_logo.svg'
import terraformLogo from '@/assets/terraform_logo.svg'
import gcsLogo from '@/assets/gcs_logo.svg'

interface CloudInfoPanelProps {
  auth: AuthStatus | null
  onTFClick: () => void
}

const providerLogos: Record<string, string> = { gcp: gcpLogo }
const providerNames: Record<string, string> = { gcp: 'Google Cloud (GCP)', aws: 'AWS', azure: 'Azure' }
const projectLabels: Record<string, string> = { gcp: 'GCP Project ID' }
const bucketLogos: Record<string, string> = { gcp: gcsLogo }

export function CloudInfoPanel({ auth, onTFClick }: CloudInfoPanelProps) {
  if (!auth) return null

  const logo = providerLogos[auth.provider]
  const displayName = providerNames[auth.provider] || auth.provider.toUpperCase()
  const projectLabel = projectLabels[auth.provider] || 'Project ID'
  const bucketLogo = bucketLogos[auth.provider]

  return (
    <div className="w-64 border-r bg-card flex flex-col">
      <div className="p-4 border-b">
        <h2 className="font-semibold text-sm uppercase tracking-wider text-muted-foreground">
          Cloud Environment
        </h2>
      </div>
      <div className="flex-1 overflow-auto p-4 space-y-6">
        {/* Provider */}
        <div className="space-y-1">
          <div className="flex items-center gap-2 text-muted-foreground">
            <span className="text-xs font-medium">Provider</span>
          </div>
          <div className="text-sm font-medium truncate flex items-center gap-2" title={displayName}>
            {logo && <img src={logo} alt={auth.provider} className="w-4 h-4" />}
            {displayName}
          </div>
        </div>

        {/* Account */}
        <div className="space-y-1">
          <div className="flex items-center gap-2 text-muted-foreground">
            <User className="w-4 h-4" />
            <span className="text-xs font-medium">Account</span>
          </div>
          <div className="text-sm font-medium truncate" title={auth.account || 'Not Authenticated'}>
            {auth.account || 'Not Authenticated'}
          </div>
        </div>

        {/* Project ID */}
        <div className="space-y-1">
          <div className="flex items-center gap-2 text-muted-foreground">
            <Hash className="w-4 h-4" />
            <span className="text-xs font-medium">{projectLabel}</span>
          </div>
          <div className="text-sm font-medium truncate" title={auth.project || 'None'}>
            {auth.project || 'None'}
          </div>
        </div>

        {/* Region */}
        <div className="space-y-1">
          <div className="flex items-center gap-2 text-muted-foreground">
            <MapPin className="w-4 h-4" />
            <span className="text-xs font-medium">Region</span>
          </div>
          <div className="text-sm font-medium truncate" title={auth.region || 'None'}>
            {auth.region || 'None'}
          </div>
        </div>

        {/* State Bucket */}
        {auth.stateBucket && (
          <div className="space-y-1">
            <div className="flex items-center gap-2 text-muted-foreground">
              {bucketLogo ? (
                <img src={bucketLogo} alt="Storage" className="w-4 h-4" />
              ) : (
                <Hash className="w-4 h-4" />
              )}
              <span className="text-xs font-medium">State Bucket</span>
            </div>
            <div className="text-sm font-medium truncate" title={auth.stateBucket}>
              {auth.stateBucket}
            </div>
          </div>
        )}

        {/* Terraform State */}
        <div className="space-y-1">
          <div className="flex items-center gap-2 text-muted-foreground">
            <img src={terraformLogo} alt="Terraform" className="w-4 h-4" />
            <span className="text-xs font-medium">Terraform State</span>
          </div>
          <button
            onClick={onTFClick}
            className="text-sm font-medium truncate text-blue-500 hover:underline text-left"
          >
            View Resources
          </button>
        </div>
      </div>
    </div>
  )
}
