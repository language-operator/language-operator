'use client'

import { createContext, useContext, useState, ReactNode, useEffect } from 'react'
import { useParams } from 'next/navigation'

interface ClusterContextType {
  selectedCluster: string | null
  setSelectedCluster: (cluster: string | null) => void
  isClusterSelected: boolean
}

const ClusterContext = createContext<ClusterContextType | undefined>(undefined)

export function ClusterProvider({ children }: { children: ReactNode }) {
  const [selectedCluster, setSelectedCluster] = useState<string | null>(null)
  const params = useParams()

  // Auto-select cluster from URL params when viewing cluster-specific pages
  useEffect(() => {
    const clusterName = params?.name as string
    if (clusterName && clusterName !== selectedCluster) {
      setSelectedCluster(clusterName)
    }
  }, [params, selectedCluster])

  const isClusterSelected = !!selectedCluster

  return (
    <ClusterContext.Provider
      value={{
        selectedCluster,
        setSelectedCluster,
        isClusterSelected,
      }}
    >
      {children}
    </ClusterContext.Provider>
  )
}

export function useClusterContext() {
  const context = useContext(ClusterContext)
  if (context === undefined) {
    throw new Error('useClusterContext must be used within a ClusterProvider')
  }
  return context
}