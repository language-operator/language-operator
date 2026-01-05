import { QuotaUsage } from './quota'

export interface OrganizationUsage extends QuotaUsage {
  organization: {
    id: string
    name: string
    namespace: string
    plan: 'free' | 'pro' | 'enterprise' | 'custom'
  }
  warnings: string[]
  isNearLimit: boolean
}

export interface UsageCardData {
  resource: string
  current: string | number
  limit: string | number
  percentage: number
  status: 'healthy' | 'warning' | 'critical' | 'limit' | 'exceeded'
  description?: string
}

export interface UsageApiResponse {
  success: boolean
  data: OrganizationUsage
}

export type UsageResourceCategory = 'resources' | 'counts'

export interface ComputeMemoryCardData {
  resource: string
  requests: {
    current: string | number
    limit: string | number
    percentage: number
    status: 'healthy' | 'warning' | 'critical' | 'limit' | 'exceeded'
  }
  limits: {
    current: string | number
    limit: string | number
    percentage: number
    status: 'healthy' | 'warning' | 'critical' | 'limit' | 'exceeded'
  }
}

export interface ResourceMetrics {
  compute: ComputeMemoryCardData
  memory: ComputeMemoryCardData
  clusters: UsageCardData
  agents: UsageCardData
  tools: UsageCardData
  models: UsageCardData
  personas: UsageCardData
}

export const USAGE_STATUS_COLORS = {
  healthy: 'bg-stone-600',
  warning: 'bg-amber-600',
  critical: 'bg-orange-600',
  limit: 'bg-amber-600',
  exceeded: 'bg-red-600'
} as const

export const USAGE_THRESHOLDS = {
  warning: 71,
  critical: 86,
  exceeded: 101  // Changed from 96 to 101 so 100% shows as 'limit' instead of 'exceeded'
} as const

export function getUsageStatus(percentage: number): 'healthy' | 'warning' | 'critical' | 'limit' | 'exceeded' {
  if (percentage >= USAGE_THRESHOLDS.exceeded) return 'exceeded'
  if (percentage === 100) return 'limit'  // Exactly at limit shows as yellow 'limit'
  if (percentage >= USAGE_THRESHOLDS.critical) return 'critical'
  if (percentage >= USAGE_THRESHOLDS.warning) return 'warning'
  return 'healthy'
}

export function formatResourceValue(value: string | undefined): string {
  if (!value || value === '0') return '0'
  
  // Handle CPU (millicores to cores)
  if (value.endsWith('m')) {
    const millicores = parseInt(value.slice(0, -1))
    return millicores >= 1000 ? `${(millicores / 1000).toFixed(1)} cores` : `${millicores}m`
  }
  
  // Handle CPU cores (convert "1" to "1 core")
  if (value.match(/^[\d.]+$/) && !value.includes('i') && !value.includes('B')) {
    const cores = parseFloat(value)
    return cores === 1 ? '1 core' : `${cores} cores`
  }
  
  // Handle Memory (bytes to readable format)
  if (value.match(/^\d+$/)) {
    const bytes = parseInt(value)
    if (bytes >= 1024 * 1024 * 1024) {
      return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)}Gi`
    } else if (bytes >= 1024 * 1024) {
      return `${(bytes / (1024 * 1024)).toFixed(1)}Mi`
    } else if (bytes >= 1024) {
      return `${(bytes / 1024).toFixed(1)}Ki`
    }
    return `${bytes}B`
  }
  
  // Return as-is for count resources and already formatted values (like "2Gi")
  return value
}

export function calculateUsagePercentage(used: string | undefined, quota: string | undefined): number {
  if (!used || !quota || quota === '0') return 0
  
  // Handle CPU values specifically
  if (used.includes('m') || quota.includes('m') || (!used.includes('i') && !used.includes('B'))) {
    // Convert CPU values to millicores for comparison
    const usedMillicores = used.endsWith('m') ? parseInt(used.slice(0, -1)) : parseFloat(used) * 1000
    const quotaMillicores = quota.endsWith('m') ? parseInt(quota.slice(0, -1)) : parseFloat(quota) * 1000
    
    if (quotaMillicores === 0) return 0
    return Math.min((usedMillicores / quotaMillicores) * 100, 100)
  }
  
  // Handle memory values (extract just the number)
  const usedNum = parseFloat(used.replace(/[^\d.]/g, ''))
  const quotaNum = parseFloat(quota.replace(/[^\d.]/g, ''))
  
  if (quotaNum === 0) return 0
  return Math.min((usedNum / quotaNum) * 100, 100)
}