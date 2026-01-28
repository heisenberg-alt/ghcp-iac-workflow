// API Types

export interface AgentStatus {
  name: string
  url: string
  status: 'online' | 'offline' | 'error'
  latency: number
}

export interface StatusResponse {
  agents: Record<string, AgentStatus>
  lastUpdated: string
}

export interface Resource {
  id: string
  type: string
  name: string
  properties: Record<string, unknown>
  line: number
}

export interface GraphNode {
  id: string
  type: string
  name: string
  category: string
  status: 'ok' | 'warning' | 'error'
  x?: number
  y?: number
  fx?: number | null
  fy?: number | null
}

export interface GraphLink {
  source: string | GraphNode
  target: string | GraphNode
  type: string
}

export interface GraphData {
  nodes: GraphNode[]
  links: GraphLink[]
}

export interface PolicyResult {
  status: string
  content: string
  violations?: PolicyViolation[]
}

export interface PolicyViolation {
  rule: string
  severity: 'CRITICAL' | 'HIGH' | 'MEDIUM' | 'LOW'
  resource: string
  message: string
}

export interface CostResult {
  status: string
  content: string
  totalMonthly?: number
  resources?: CostResource[]
}

export interface CostResource {
  type: string
  name: string
  monthlyCost: number
  sku: string
}

export interface AnalysisResult {
  requestId: string
  status: string
  results: {
    policy?: PolicyResult
    cost?: CostResult
  }
  resources: Resource[]
  graph: GraphData
}

export interface WSMessage {
  type: string
  payload: unknown
}

export interface ChatMessage {
  id: string
  role: 'user' | 'assistant'
  content: string
  timestamp: Date
}
