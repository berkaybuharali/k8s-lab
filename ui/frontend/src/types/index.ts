export interface AuthStatus {
  authenticated: boolean
  account?: string
  project?: string
  region?: string
  provider: string
  stateBucket?: string
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

export interface K8sNode {
  metadata: { 
    name: string
    labels: Record<string, string> 
  }
  status: {
    addresses: { type: string; address: string }[]
    conditions: { type: string; status: string }[]
    nodeInfo: { osImage: string; kernelVersion: string; kubeletVersion: string }
  }
}

export interface K8sPod {
  metadata: { 
    name: string
    namespace: string
    creationTimestamp: string 
  }
  status: { 
    phase: string
    startTime: string
    podIP?: string
    containerStatuses?: { restartCount: number }[] 
  }
  spec: { 
    nodeName: string
    containers: { image: string }[]
  }
}

export interface K8sPVC {
  metadata: { name: string; namespace: string }
  spec: { resources: { requests: { storage: string } } }
  status: { phase: string }
}

export interface VeleroBackup {
  metadata: { name: string }
  status: { phase: string; expiration: string; completionTimestamp: string }
}