'use client'

import { useParams } from 'next/navigation'
import { useModel } from '@/hooks/use-models'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { ResourceEventsActivity } from '@/components/ui/events-activity'
import { AlertCircle, CheckCircle, Clock } from 'lucide-react'
import { LanguageModel } from '@/types/model'


interface ModelOverviewProps {
  model: LanguageModel
  clusterName: string
}

function ModelOverview({ model, clusterName }: ModelOverviewProps) {
  return (
    <div className="space-y-6">
      {/* Basic Info */}
      <Card>
        <CardHeader>
          <CardTitle>Basic Information</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div>
              <p className="text-sm font-medium text-stone-600 dark:text-stone-400">Provider</p>
              <Badge variant="outline">{model.spec.provider}</Badge>
            </div>
            <div>
              <p className="text-sm font-medium text-stone-600 dark:text-stone-400">Model Name</p>
              <p className="text-sm font-mono">{model.spec.modelName}</p>
            </div>
            {model.spec.endpoint && (
              <div>
                <p className="text-sm font-medium text-stone-600 dark:text-stone-400">Endpoint</p>
                <p className="text-sm font-mono text-xs break-all">{model.spec.endpoint}</p>
              </div>
            )}
          </div>
        </CardContent>
      </Card>

      {/* Status & Conditions */}
      <Card>
        <CardHeader>
          <CardTitle>Status & Conditions</CardTitle>
        </CardHeader>
        <CardContent>
          {model.status?.conditions && model.status.conditions.length > 0 ? (
            <div className="space-y-3">
              {model.status.conditions.map((condition, index) => (
                <div key={index} className="flex items-center justify-between p-3 border border-stone-200 dark:border-stone-700">
                  <div>
                    <p className="text-sm font-medium">{condition.type}</p>
                    {condition.message && (
                      <p className="text-xs text-stone-600 dark:text-stone-400">{condition.message}</p>
                    )}
                  </div>
                  <div className="flex items-center space-x-2">
                    {condition.status === 'True' ? (
                      <CheckCircle className="h-4 w-4 text-green-500" />
                    ) : condition.status === 'False' ? (
                      <AlertCircle className="h-4 w-4 text-red-500" />
                    ) : (
                      <Clock className="h-4 w-4 text-yellow-500" />
                    )}
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <p className="text-sm text-stone-600 dark:text-stone-400">No status conditions available</p>
          )}
        </CardContent>
      </Card>

      {/* Events */}
      <ResourceEventsActivity
        resourceType="model"
        resourceName={model.metadata.name!}
        namespace={model.metadata.namespace!}
        clusterName={clusterName}
      />


      {/* Rate Limits */}
      {model.spec.rateLimits && (
        <Card>
          <CardHeader>
            <CardTitle>Rate Limits</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid gap-4 md:grid-cols-3">
              {model.spec.rateLimits.requestsPerMinute && (
                <div>
                  <p className="text-sm font-medium text-stone-600 dark:text-stone-400">Requests per Minute</p>
                  <p className="text-sm">{model.spec.rateLimits.requestsPerMinute.toLocaleString()}</p>
                </div>
              )}
              {model.spec.rateLimits.tokensPerMinute && (
                <div>
                  <p className="text-sm font-medium text-stone-600 dark:text-stone-400">Tokens per Minute</p>
                  <p className="text-sm">{model.spec.rateLimits.tokensPerMinute.toLocaleString()}</p>
                </div>
              )}
              {model.spec.rateLimits.concurrentRequests && (
                <div>
                  <p className="text-sm font-medium text-stone-600 dark:text-stone-400">Concurrent Requests</p>
                  <p className="text-sm">{model.spec.rateLimits.concurrentRequests}</p>
                </div>
              )}
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  )
}

export default function ModelOverviewPage() {
  const params = useParams()
  const clusterName = params.name as string
  const modelName = params.modelName as string

  const { data: modelResponse, isLoading } = useModel(modelName, clusterName)
  const model = modelResponse?.data

  if (isLoading) {
    return (
      <div className="space-y-6">
        <Card>
          <CardContent className="flex items-center justify-center py-16">
            <div className="text-center">
              <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-gray-900 mx-auto mb-4"></div>
              <p className="text-gray-600">Loading model...</p>
            </div>
          </CardContent>
        </Card>
      </div>
    )
  }

  if (!model) {
    return null // Layout handles error state
  }

  return <ModelOverview model={model} clusterName={clusterName} />
}