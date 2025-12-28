'use client'

import { useParams } from 'next/navigation'
import { useModel } from '@/hooks/use-models'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Settings, Database, Shield, RefreshCw, Globe } from 'lucide-react'
import { LanguageModel } from '@/types/model'

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
      {model.spec.caching && (
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
                <Badge variant={model.spec.caching.enabled ? "default" : "secondary"}>
                  {model.spec.caching.enabled ? "Enabled" : "Disabled"}
                </Badge>
              </div>
              {model.spec.caching.ttl && (
                <div>
                  <p className="text-sm font-medium text-stone-600 dark:text-stone-400">TTL (Time to Live)</p>
                  <p className="text-sm">{model.spec.caching.ttl}</p>
                </div>
              )}
            </div>
          </CardContent>
        </Card>
      )}

      {/* Rate Limits */}
      {model.spec.rateLimits && (
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
            </div>
          </CardContent>
        </Card>
      )}

      {/* Retry Policy */}
      {model.spec.retryPolicy && (
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
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {(model.spec.retryPolicy as any).maxAttempts && (
                <div>
                  <p className="text-sm font-medium text-stone-600 dark:text-stone-400">Max Attempts</p>
                  <p className="text-sm">{(model.spec.retryPolicy as any).maxAttempts}</p>
                </div>
              )}
              {(model.spec.retryPolicy as any).initialBackoff && (
                <div>
                  <p className="text-sm font-medium text-stone-600 dark:text-stone-400">Initial Backoff</p>
                  <p className="text-sm">{(model.spec.retryPolicy as any).initialBackoff}</p>
                </div>
              )}
              {(model.spec.retryPolicy as any).maxBackoff && (
                <div>
                  <p className="text-sm font-medium text-stone-600 dark:text-stone-400">Max Backoff</p>
                  <p className="text-sm">{(model.spec.retryPolicy as any).maxBackoff}</p>
                </div>
              )}
              {(model.spec.retryPolicy as any).backoffMultiplier && (
                <div>
                  <p className="text-sm font-medium text-stone-600 dark:text-stone-400">Backoff Multiplier</p>
                  <p className="text-sm">{(model.spec.retryPolicy as any).backoffMultiplier}</p>
                </div>
              )}
            </div>
          </CardContent>
        </Card>
      )}

      {/* Network Policy */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Globe className="h-5 w-5" />
            Network Policy
          </CardTitle>
          <CardDescription>
            External network access rules
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {model.spec.egress && model.spec.egress.length > 0 ? (
            <div className="space-y-3">
              {model.spec.egress.map((rule, index) => (
                <div key={index} className="border border-stone-200 p-3 dark:border-stone-700">
                  <div className="space-y-2">
                    {rule.description && (
                      <div>
                        <p className="text-sm font-medium text-stone-600 dark:text-stone-400">Description</p>
                        <p className="text-sm">{rule.description}</p>
                      </div>
                    )}
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-xs">
                      {rule.to?.dns && rule.to.dns.length > 0 && (
                        <div>
                          <p className="text-sm font-medium text-stone-600 dark:text-stone-400">DNS Names</p>
                          <p className="text-sm font-mono">{rule.to.dns.join(', ')}</p>
                        </div>
                      )}
                      {rule.ports && rule.ports.length > 0 && (
                        <div>
                          <p className="text-sm font-medium text-stone-600 dark:text-stone-400">Ports</p>
                          <p className="text-sm font-mono">{rule.ports.map(p => p.port).join(', ')}</p>
                        </div>
                      )}
                    </div>
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <div className="text-center py-6">
              <Globe className="h-8 w-8 text-stone-500 dark:text-stone-400 mx-auto mb-2" />
              <p className="text-sm text-stone-600 dark:text-stone-400">No network egress rules configured</p>
              <p className="text-xs text-stone-500 dark:text-stone-500 mt-1">
                Network policies control external access from the model
              </p>
            </div>
          )}
        </CardContent>
      </Card>
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