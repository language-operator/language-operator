'use client'

import { useParams } from 'next/navigation'
import { useModel } from '@/hooks/use-models'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Settings, Database, Shield, RefreshCw } from 'lucide-react'
import { LanguageModel } from '@/types/model'
import { formatCurrencyAutoPrecision } from '@/lib/currency'

interface ModelDetailsProps {
  model: LanguageModel
}


function ModelDetails({ model }: ModelDetailsProps) {
  return (
    <div className="space-y-6">
      {/* Model Configuration */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Settings className="h-5 w-5" />
            Model Configuration
          </CardTitle>
          <CardDescription>
            LLM parameters and behavior settings
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <p className="text-sm font-medium text-stone-600 dark:text-stone-400">Temperature</p>
              <p className="text-sm">{model.spec.configuration?.temperature ?? 0.7}</p>
            </div>
            <div>
              <p className="text-sm font-medium text-stone-600 dark:text-stone-400">Max Tokens</p>
              <p className="text-sm">{(model.spec.configuration?.maxTokens ?? 4096).toLocaleString()}</p>
            </div>
            <div>
              <p className="text-sm font-medium text-stone-600 dark:text-stone-400">Top P</p>
              <p className="text-sm">{model.spec.configuration?.topP ?? 1.0}</p>
            </div>
            <div>
              <p className="text-sm font-medium text-stone-600 dark:text-stone-400">Context Window</p>
              <p className="text-sm">{((model.spec.configuration as any)?.contextWindow ?? 8192).toLocaleString()} tokens</p>
            </div>
            {model.spec.configuration?.frequencyPenalty !== undefined && (
              <div>
                <p className="text-sm font-medium text-stone-600 dark:text-stone-400">Frequency Penalty</p>
                <p className="text-sm">{model.spec.configuration.frequencyPenalty}</p>
              </div>
            )}
            {model.spec.configuration?.presencePenalty !== undefined && (
              <div>
                <p className="text-sm font-medium text-stone-600 dark:text-stone-400">Presence Penalty</p>
                <p className="text-sm">{model.spec.configuration.presencePenalty}</p>
              </div>
            )}
            {(model.spec as any).timeout && (
              <div>
                <p className="text-sm font-medium text-stone-600 dark:text-stone-400">Request Timeout</p>
                <p className="text-sm">{(model.spec as any).timeout}</p>
              </div>
            )}
          </div>
          {model.spec.configuration?.stopSequences && model.spec.configuration.stopSequences.length > 0 && (
            <div>
              <p className="text-sm font-medium text-stone-600 dark:text-stone-400">Stop Sequences</p>
              <div className="flex flex-wrap gap-1 mt-1">
                {model.spec.configuration.stopSequences.map((seq, index) => (
                  <Badge key={index} variant="outline" className="text-xs font-mono">
                    {seq}
                  </Badge>
                ))}
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Caching */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Database className="h-5 w-5" />
            Response Caching
          </CardTitle>
          <CardDescription>
            Caching configuration for improved performance
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <p className="text-sm font-medium text-stone-600 dark:text-stone-400">Status</p>
              <Badge variant={model.spec.caching?.enabled ? "default" : "secondary"}>
                {model.spec.caching?.enabled ? "Enabled" : "Disabled"}
              </Badge>
            </div>
            {model.spec.caching?.ttl && (
              <div>
                <p className="text-sm font-medium text-stone-600 dark:text-stone-400">TTL (Time to Live)</p>
                <p className="text-sm">{model.spec.caching.ttl}</p>
              </div>
            )}
          </div>
          {!model.spec.caching && (
            <p className="text-sm text-muted-foreground italic">Off</p>
          )}
        </CardContent>
      </Card>

      {/* Rate Limits */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Shield className="h-5 w-5" />
            Rate Limiting
          </CardTitle>
          <CardDescription>
            Request rate limits and throttling
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {model.spec.rateLimits ? (
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
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
              {!model.spec.rateLimits.requestsPerMinute && !model.spec.rateLimits.tokensPerMinute && !model.spec.rateLimits.concurrentRequests && (
                <p className="text-sm text-muted-foreground italic">Off</p>
              )}
            </div>
          ) : (
            <p className="text-sm text-muted-foreground italic">Off</p>
          )}
        </CardContent>
      </Card>

      {/* Retry Policy */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <RefreshCw className="h-5 w-5" />
            Retry Policy
          </CardTitle>
          <CardDescription>
            Automatic retry configuration for failed requests
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {model.spec.retryPolicy ? (
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {(model.spec.retryPolicy as any).maxAttempts ? (
                <div>
                  <p className="text-sm font-medium text-stone-600 dark:text-stone-400">Max Attempts</p>
                  <p className="text-sm">{(model.spec.retryPolicy as any).maxAttempts}</p>
                </div>
              ) : (
                <p className="text-sm text-muted-foreground italic">Off</p>
              )}
            </div>
          ) : (
            <p className="text-sm text-muted-foreground italic">Off</p>
          )}
        </CardContent>
      </Card>

      {/* Cost Tracking */}
      {model.spec.costTracking && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Settings className="h-5 w-5" />
              Cost Tracking
            </CardTitle>
            <CardDescription>
              Token cost configuration and tracking
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              {model.spec.costTracking.inputTokenCost && (
                <div>
                  <p className="text-sm font-medium text-stone-600 dark:text-stone-400">Input Token Cost</p>
                  <p className="text-sm">{formatCurrencyAutoPrecision(model.spec.costTracking.inputTokenCost, model.spec.costTracking.currency)} per 1,000 tokens</p>
                </div>
              )}
              {model.spec.costTracking.outputTokenCost && (
                <div>
                  <p className="text-sm font-medium text-stone-600 dark:text-stone-400">Output Token Cost</p>
                  <p className="text-sm">{formatCurrencyAutoPrecision(model.spec.costTracking.outputTokenCost, model.spec.costTracking.currency)} per 1,000 tokens</p>
                </div>
              )}
              {model.spec.costTracking.currency && (
                <div>
                  <p className="text-sm font-medium text-stone-600 dark:text-stone-400">Currency</p>
                  <p className="text-sm">{model.spec.costTracking.currency}</p>
                </div>
              )}
            </div>
          </CardContent>
        </Card>
      )}

    </div>
  )
}

export default function ModelDetailsPage() {
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
              <p className="text-gray-600">Loading model details...</p>
            </div>
          </CardContent>
        </Card>
      </div>
    )
  }

  if (!model) {
    return null // Layout handles error state
  }

  return <ModelDetails model={model} />
}