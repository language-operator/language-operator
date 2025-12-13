'use client'

import React from 'react'
import { useParams } from 'next/navigation'
import { AuthenticatedLayout } from '@/components/layout/authenticated-layout'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Cpu, Plus, ExternalLink } from 'lucide-react'
import Link from 'next/link'
import { useModels } from '@/hooks/use-models'

export default function ClusterModels() {
  const params = useParams()
  const clusterName = params?.name as string
  
  // For now, fetch all models and filter client-side
  // TODO: Update API to support cluster/namespace filtering
  const { data: modelsResponse, isLoading, error } = useModels({ limit: 100 })
  const allModels = modelsResponse?.data || []
  
  // Debug logging
  console.log('Models response:', modelsResponse)
  console.log('All models:', allModels)
  console.log('Error:', error)
  console.log('Is loading:', isLoading)
  
  // Manual test API call
  React.useEffect(() => {
    console.log('Models page mounted, triggering test API call...')
    fetch('/api/models?limit=5')
      .then(res => {
        console.log('Manual API response status:', res.status)
        return res.json()
      })
      .then(data => {
        console.log('Manual API response data:', data)
      })
      .catch(err => {
        console.error('Manual API call error:', err)
      })
  }, [])
  
  // For now, show all models (TODO: implement proper cluster-scoped filtering)
  const clusterModels = allModels

  const getStatusColor = (phase?: string) => {
    switch (phase) {
      case 'Ready':
      case 'Available':
        return 'bg-green-100 text-green-800'
      case 'Pending':
        return 'bg-yellow-100 text-yellow-800'
      case 'Failed':
        return 'bg-red-100 text-red-800'
      default:
        return 'bg-gray-100 text-gray-800'
    }
  }

  if (isLoading) {
    return (
      <AuthenticatedLayout>
        <div className="flex items-center justify-center h-64">
          <div className="text-lg">Loading models...</div>
        </div>
      </AuthenticatedLayout>
    )
  }

  return (
    <AuthenticatedLayout>
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold">Models</h1>
            <p className="text-gray-600 mt-1">
              Language models in the {clusterName} cluster ({clusterModels.length} models)
            </p>
          </div>
          <Button asChild>
            <Link href={`/clusters/${clusterName}/models/new`}>
              <Plus className="h-4 w-4 mr-2" />
              New Model
            </Link>
          </Button>
        </div>

        {/* Models List or Empty State */}
        {clusterModels.length === 0 ? (
          <Card>
            <CardContent className="flex flex-col items-center justify-center py-16">
              <Cpu className="h-16 w-16 text-gray-400 mb-4" />
              <CardTitle className="text-xl mb-2">No models yet</CardTitle>
              <CardDescription className="text-center max-w-md mb-6">
                Language models define the AI capabilities available in this cluster. 
                Add your first model to get started.
              </CardDescription>
              <Button asChild>
                <Link href={`/clusters/${clusterName}/models/new`}>
                  <Plus className="h-4 w-4 mr-2" />
                  Create Your First Model
                </Link>
              </Button>
            </CardContent>
          </Card>
        ) : (
          <div className="grid gap-4">
            {clusterModels.map((model: any) => (
              <Card key={model.metadata.name}>
                <CardContent className="p-6">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-4">
                      <Cpu className="h-8 w-8 text-blue-500" />
                      <div>
                        <h3 className="font-semibold text-lg">{model.metadata.name}</h3>
                        <p className="text-sm text-gray-600">
                          {model.spec.provider} • {model.spec.modelName}
                        </p>
                        {model.spec.endpoint && (
                          <p className="text-xs text-gray-500 font-mono">
                            {model.spec.endpoint}
                          </p>
                        )}
                      </div>
                    </div>
                    <div className="flex items-center gap-3">
                      <Badge className={getStatusColor(model.status?.phase)}>
                        {model.status?.phase || 'Unknown'}
                      </Badge>
                      <Button variant="outline" size="sm" asChild>
                        <Link href={`/clusters/${clusterName}/models/${model.metadata.name}`}>
                          View Details
                          <ExternalLink className="h-3 w-3 ml-1" />
                        </Link>
                      </Button>
                    </div>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        )}
      </div>
    </AuthenticatedLayout>
  )
}