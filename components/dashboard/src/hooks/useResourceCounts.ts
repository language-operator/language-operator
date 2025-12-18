import { useState, useEffect } from 'react'
import { fetchWithOrganization } from '@/lib/api-client'
import { useOrganizationStore } from '@/store/organization-store'

export interface ResourceCounts {
  agents: number
  models: number
  tools: number
  personas: number
  clusters: number
}

export interface UseResourceCountsReturn {
  counts: ResourceCounts | null
  loading: boolean
  error: string | null
  refetch: () => Promise<void>
}

export function useResourceCounts(clusterName?: string): UseResourceCountsReturn {
  const [counts, setCounts] = useState<ResourceCounts | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const { activeOrganizationId } = useOrganizationStore()

  const fetchCounts = async () => {
    try {
      setLoading(true)
      setError(null)

      // Use cluster-specific endpoint if cluster name is provided
      const endpoint = clusterName 
        ? `/api/clusters/${clusterName}/counts`
        : '/api/dashboard/counts'
      
      const response = await fetchWithOrganization(endpoint)
      
      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}))
        throw new Error(errorData.error || `HTTP ${response.status}: ${response.statusText}`)
      }

      const result = await response.json()
      
      if (!result.success || !result.data) {
        throw new Error('Invalid response format from server')
      }

      setCounts(result.data)
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to fetch resource counts'
      setError(errorMessage)
      console.error('Error fetching resource counts:', err)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchCounts()
  }, [clusterName, activeOrganizationId])

  return {
    counts,
    loading,
    error,
    refetch: fetchCounts,
  }
}