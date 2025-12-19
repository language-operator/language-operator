import { useState, useEffect, useRef, useCallback } from 'react'
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
  isStale: boolean // Indicates if we're showing stale data while revalidating
}

// Cache for storing resource counts with timestamps
interface CacheEntry {
  data: ResourceCounts
  timestamp: number
  isStale: boolean
}

const countsCache = new Map<string, CacheEntry>()
const STALE_TIME = 30 * 1000 // 30 seconds - show stale data after this
const CACHE_TIME = 5 * 60 * 1000 // 5 minutes - completely remove after this

export function useResourceCounts(clusterName?: string): UseResourceCountsReturn {
  const [counts, setCounts] = useState<ResourceCounts | null>(null)
  const [loading, setLoading] = useState(true)
  const [isStale, setIsStale] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const { activeOrganizationId } = useOrganizationStore()
  const abortControllerRef = useRef<AbortController | null>(null)
  
  const cacheKey = `${activeOrganizationId}:${clusterName || 'dashboard'}`

  const fetchCounts = useCallback(async (showStaleData = true) => {
    try {
      // Cancel any ongoing request
      if (abortControllerRef.current) {
        abortControllerRef.current.abort()
      }
      
      abortControllerRef.current = new AbortController()
      
      // Check cache first
      const cached = countsCache.get(cacheKey)
      const now = Date.now()
      
      if (cached && showStaleData) {
        const age = now - cached.timestamp
        
        if (age < STALE_TIME) {
          // Fresh data - use immediately
          setCounts(cached.data)
          setIsStale(false)
          setLoading(false)
          setError(null)
          return
        } else if (age < CACHE_TIME) {
          // Stale but valid - show immediately and fetch in background
          setCounts(cached.data)
          setIsStale(true)
          setLoading(false)
          setError(null)
          // Continue to fetch fresh data below
        } else {
          // Expired - remove from cache
          countsCache.delete(cacheKey)
        }
      }
      
      if (!cached || !showStaleData) {
        setLoading(true)
        setError(null)
      }

      // Use cluster-specific endpoint if cluster name is provided
      const endpoint = clusterName 
        ? `/api/clusters/${clusterName}/counts`
        : '/api/dashboard/counts'
      
      const response = await fetchWithOrganization(endpoint, {
        signal: abortControllerRef.current.signal
      })
      
      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}))
        throw new Error(errorData.error || `HTTP ${response.status}: ${response.statusText}`)
      }

      const result = await response.json()
      
      if (!result.success || !result.data) {
        throw new Error('Invalid response format from server')
      }

      // Update cache
      countsCache.set(cacheKey, {
        data: result.data,
        timestamp: now,
        isStale: false
      })
      
      setCounts(result.data)
      setIsStale(false)
      setError(null)
    } catch (err) {
      if (err instanceof Error && err.name === 'AbortError') {
        // Request was cancelled, ignore
        return
      }
      
      const errorMessage = err instanceof Error ? err.message : 'Failed to fetch resource counts'
      
      // Only show error if we don't have stale data to fall back to
      if (!counts) {
        setError(errorMessage)
      }
      console.error('Error fetching resource counts:', err)
    } finally {
      setLoading(false)
    }
  }, [clusterName, activeOrganizationId, cacheKey, counts])

  useEffect(() => {
    fetchCounts(true)
    
    // Cleanup on unmount
    return () => {
      if (abortControllerRef.current) {
        abortControllerRef.current.abort()
      }
    }
  }, [fetchCounts])

  return {
    counts,
    loading,
    error,
    isStale,
    refetch: () => fetchCounts(false),
  }
}