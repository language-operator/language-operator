'use client'

import { useState } from 'react'
import { useRouter } from 'next/navigation'
import { useClusters } from '@/hooks/use-clusters'
import { useOrganization } from '@/components/organization-provider'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Boxes, ExternalLink, Users } from 'lucide-react'
import { LanguageCluster } from '@/types/cluster'

interface ClusterSelectionModalProps {
  isOpen: boolean
  onClose: () => void
  onClusterSelect: (clusterName: string) => void
  actionType: 'agent' | 'model' | 'tool'
  actionTitle: string
  actionDescription: string
}

export function ClusterSelectionModal({
  isOpen,
  onClose,
  onClusterSelect,
  actionType,
  actionTitle,
  actionDescription,
}: ClusterSelectionModalProps) {
  const router = useRouter()
  const { getOrgUrl } = useOrganization()
  const [selectedCluster, setSelectedCluster] = useState<string | null>(null)
  const { data: clustersData, isLoading, error } = useClusters({ limit: 100 })
  const clusters = clustersData?.data || []

  const handleContinue = () => {
    if (selectedCluster) {
      onClusterSelect(selectedCluster)
      onClose()
      setSelectedCluster(null) // Reset for next time
    }
  }

  const handleCancel = () => {
    onClose()
    setSelectedCluster(null) // Reset selection
  }

  const getActionIcon = () => {
    switch (actionType) {
      case 'agent':
        return <Users className="h-5 w-5 text-blue-500" />
      case 'model':
        return <Boxes className="h-5 w-5 text-green-500" />
      case 'tool':
        return <ExternalLink className="h-5 w-5 text-purple-500" />
    }
  }

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            {getActionIcon()}
            {actionTitle}
          </DialogTitle>
          <DialogDescription>
            {actionDescription} Please select a cluster to continue.
          </DialogDescription>
        </DialogHeader>

        <div className="py-4">
          {isLoading ? (
            <div className="flex items-center justify-center py-8">
              <div className="text-sm text-gray-500">Loading clusters...</div>
            </div>
          ) : error ? (
            <div className="flex items-center justify-center py-8">
              <div className="text-sm text-red-500">Error loading clusters</div>
            </div>
          ) : clusters.length === 0 ? (
            <div className="text-center py-8">
              <Boxes className="h-12 w-12 text-gray-300 mx-auto mb-4" />
              <h3 className="text-lg font-medium text-gray-900 mb-2">No clusters available</h3>
              <p className="text-sm text-gray-500 mb-4">
                You need to create a cluster before you can {actionType === 'agent' ? 'deploy agents' : actionType === 'model' ? 'add models' : 'configure tools'}.
              </p>
              <Button 
                onClick={() => router.push(getOrgUrl('/clusters/new'))}
                variant="outline"
                className="gap-2"
              >
                <Boxes className="h-4 w-4" />
                Create First Cluster
              </Button>
            </div>
          ) : (
            <div className="space-y-3">
              <div className="text-sm font-medium text-gray-700 mb-3">
                Available Clusters ({clusters.length})
              </div>
              <div className="grid gap-2 max-h-64 overflow-y-auto">
                {clusters.map((cluster: LanguageCluster) => (
                  <div
                    key={cluster.metadata?.name}
                    className={`p-3 border rounded-lg cursor-pointer transition-all ${
                      selectedCluster === cluster.metadata?.name
                        ? 'border-blue-500 bg-blue-50'
                        : 'border-gray-200 hover:border-gray-300 hover:bg-gray-50'
                    }`}
                    onClick={() => setSelectedCluster(cluster.metadata?.name || '')}
                  >
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-3">
                        <Boxes className="h-5 w-5 text-blue-500" />
                        <div>
                          <div className="font-medium text-sm">
                            {cluster.metadata?.name}
                          </div>
                          {cluster.spec?.domain && (
                            <div className="text-xs text-gray-500">
                              {cluster.spec.domain}
                            </div>
                          )}
                        </div>
                      </div>
                      <div className="flex items-center gap-2">
                        {cluster.status?.phase && (
                          <span className={`text-xs px-2 py-1 rounded-full ${
                            cluster.status.phase === 'Ready' 
                              ? 'bg-green-100 text-green-800' 
                              : cluster.status.phase === 'Pending'
                              ? 'bg-yellow-100 text-yellow-800'
                              : 'bg-gray-100 text-gray-800'
                          }`}>
                            {cluster.status.phase}
                          </span>
                        )}
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={handleCancel}>
            Cancel
          </Button>
          <Button 
            onClick={handleContinue} 
            disabled={!selectedCluster || clusters.length === 0}
          >
            Continue with {selectedCluster || 'Selected Cluster'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}