export interface AuthStatus {
  authenticated: boolean
  account?: string
  project?: string
  region?: string
  provider: string
  error?: string
}

export interface GlobalStatus {
  infra: string
  k8s: string
  tools: string
  apps: string
  tunnel: string
  version?: string
}

export interface LogMessage {
  type: 'log' | 'error' | 'done' | 'start'
  data: string
  timestamp: string
}