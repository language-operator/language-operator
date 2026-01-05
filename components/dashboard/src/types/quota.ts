export interface OrganizationQuota {
  'count/languageagents': string
  'count/languagemodels': string
  'count/languagetools': string
  'count/languagepersonas': string
  'count/languageclusters': string
  'count/members': string
  'requests.cpu': string
  'requests.memory': string
  'limits.cpu': string
  'limits.memory': string
}

export interface QuotaUsage {
  quota: OrganizationQuota
  used: Partial<OrganizationQuota>
  available: Partial<OrganizationQuota>
  percentUsed: Record<string, number>
}

export interface QuotaUpdateRequest {
  plan?: 'free' | 'pro' | 'enterprise'
  quotas?: OrganizationQuota
}

export const QUOTA_FIELDS = [
  'count/languageagents',
  'count/languagemodels',
  'count/languagetools',
  'count/languagepersonas',
  'count/languageclusters',
  'count/members',
  'requests.cpu',
  'requests.memory',
  'limits.cpu',
  'limits.memory'
] as const

export const QUOTA_LABELS: Record<string, string> = {
  'count/languageagents': 'Agents',
  'count/languagemodels': 'Models',
  'count/languagetools': 'Tools',
  'count/languagepersonas': 'Personas',
  'count/languageclusters': 'Clusters',
  'count/members': 'Members',
  'requests.cpu': 'CPU Requests',
  'requests.memory': 'Memory Requests',
  'limits.cpu': 'CPU Limits',
  'limits.memory': 'Memory Limits'
}

export const QUOTA_DESCRIPTIONS: Record<string, string> = {
  'count/languageagents': 'Maximum number of agents',
  'count/languagemodels': 'Maximum number of model configurations',
  'count/languagetools': 'Maximum number of tools',
  'count/languagepersonas': 'Maximum number of personas',
  'count/languageclusters': 'Maximum number of clusters',
  'count/members': 'Maximum number of organization members',
  'requests.cpu': 'CPU requests (e.g., 1000m or 1)',
  'requests.memory': 'Memory requests (e.g., 2Gi or 1024Mi)',
  'limits.cpu': 'CPU limits (e.g., 2000m or 2)',
  'limits.memory': 'Memory limits (e.g., 4Gi or 2048Mi)'
}
