'use client'

import { useParams, useRouter } from 'next/navigation'
import { ModelForm, ModelFormData } from '@/components/forms/model-form'
import { useModel, useUpdateModel } from '@/hooks/use-models'
import { Skeleton } from '@/components/ui/skeleton'
import { useOrganization } from '@/components/organization-provider'

export default function ClusterEditModelPage() {
  const params = useParams()
  const router = useRouter()
  const clusterName = params.name as string
  const modelName = params.modelName as string
  const { getOrgUrl } = useOrganization()
  const { data: modelResponse, isLoading: isLoadingModel } = useModel(modelName, clusterName)
  const model = modelResponse?.data
  
  const updateModel = useUpdateModel(clusterName)

  const handleSubmit = async (formData: ModelFormData) => {
    try {
      await updateModel.mutateAsync({
        modelName,
        updateData: {
          name: formData.name,
          provider: formData.provider,
          model: formData.model,
          endpoint: formData.endpoint,
          apiKey: formData.apiKey || undefined,
          description: formData.description || undefined,
          spec: {
            provider: formData.provider,
            model: formData.model,
            endpoint: formData.endpoint,
            apiKey: formData.apiKey || undefined,
            timeout: formData.timeout,
            parameters: {
              maxTokens: formData.maxTokens,
              temperature: formData.temperature,
              topP: formData.topP,
              frequencyPenalty: formData.frequencyPenalty,
              presencePenalty: formData.presencePenalty,
              additionalParameters: formData.additionalParameters,
            },
            contextWindow: formData.contextWindow,
            costTracking: {
              inputTokenCost: formData.costPerInputToken,
              outputTokenCost: formData.costPerOutputToken,
              currency: formData.currency || 'USD',
              enabled: formData.costTrackingEnabled || false
            },
            enabled: formData.enabled,
            requireApproval: formData.requireApproval,
            // Enterprise features
            caching: {
              enabled: formData.cachingEnabled || false,
              backend: formData.cachingBackend,
              ttl: formData.cachingTtl,
              maxSize: formData.cachingMaxSize
            },
            loadBalancing: formData.loadBalancingEnabled ? {
              strategy: formData.loadBalancingStrategy,
              endpoints: formData.loadBalancingEndpoints,
              healthCheck: formData.healthCheckEnabled ? {
                interval: formData.healthCheckInterval,
                timeout: formData.healthCheckTimeout,
                healthyThreshold: formData.healthCheckHealthyThreshold,
                unhealthyThreshold: formData.healthCheckUnhealthyThreshold
              } : undefined
            } : undefined,
            observability: {
              logging: {
                level: formData.logLevel,
                logRequests: formData.logRequests,
                logResponses: formData.logResponses
              },
              metrics: formData.metricsEnabled,
              tracing: formData.tracingEnabled
            },
            rateLimiting: {
              requestsPerMinute: formData.requestsPerMinute,
              tokensPerMinute: formData.tokensPerMinute,
              concurrentRequests: formData.concurrentRequests
            },
            retryPolicy: {
              maxAttempts: formData.retryMaxAttempts,
              initialBackoff: formData.retryInitialBackoff,
              maxBackoff: formData.retryMaxBackoff,
              backoffMultiplier: formData.retryBackoffMultiplier,
              retryableStatusCodes: formData.retryableStatusCodes
            },
            egress: formData.egress?.map((rule: any) => ({
              ...rule,
              ports: rule.ports?.map((port: any) => 
                typeof port === 'number' ? { port, protocol: 'TCP' } : port
              )
            })),
            regions: formData.regions,
            fallbacks: formData.fallbacks
          },
        }
      })

      // Redirect to model detail page
      router.push(getOrgUrl(`/clusters/${clusterName}/models/${modelName}`))
    } catch (err: any) {
      console.error('Error updating model:', err)
      // Error is handled by the mutation hook
    }
  }

  const handleCancel = () => {
    router.push(`/clusters/${clusterName}/models/${modelName}`)
  }

  // Convert model data to form data format
  const initialData: Partial<ModelFormData> | undefined = model ? {
    name: model.metadata.name,
    provider: model.spec.provider,
    model: model.spec.modelName,
    endpoint: model.spec.endpoint,
    description: model.spec.description || '',
    maxTokens: model.spec.configuration?.maxTokens || 4096,
    temperature: model.spec.configuration?.temperature || 0.7,
    topP: model.spec.configuration?.topP || 1.0,
    frequencyPenalty: model.spec.configuration?.frequencyPenalty || 0.0,
    presencePenalty: model.spec.configuration?.presencePenalty || 0.0,
    contextWindow: model.spec.configuration?.contextWindow || 8192,
    costPerInputToken: model.spec.costTracking?.inputTokenCost || 0.0,
    costPerOutputToken: model.spec.costTracking?.outputTokenCost || 0.0,
    costTrackingEnabled: model.spec.costTracking?.enabled || false,
    enabled: model.spec.enabled !== false,
    requireApproval: model.spec.requireApproval || false,
    
    // Enterprise features from existing model
    timeout: model.spec.timeout || '5m',
    additionalParameters: model.spec.configuration?.additionalParameters || {},
    cachingEnabled: model.spec.caching?.enabled || false,
    cachingBackend: model.spec.caching?.backend || 'memory',
    cachingTtl: model.spec.caching?.ttl || '5m',
    cachingMaxSize: model.spec.caching?.maxSize || 100,
    currency: model.spec.costTracking?.currency || 'USD',
    loadBalancingEnabled: !!model.spec.loadBalancing,
    loadBalancingStrategy: model.spec.loadBalancing?.strategy || 'round-robin',
    loadBalancingEndpoints: model.spec.loadBalancing?.endpoints || [],
    healthCheckEnabled: !!model.spec.loadBalancing?.healthCheck,
    healthCheckInterval: model.spec.loadBalancing?.healthCheck?.interval || '30s',
    healthCheckTimeout: model.spec.loadBalancing?.healthCheck?.timeout || '5s',
    healthCheckHealthyThreshold: model.spec.loadBalancing?.healthCheck?.healthyThreshold || 2,
    healthCheckUnhealthyThreshold: model.spec.loadBalancing?.healthCheck?.unhealthyThreshold || 3,
    fallbacks: model.spec.fallbacks || [],
    logLevel: model.spec.observability?.logging?.level || 'info',
    logRequests: model.spec.observability?.logging?.logRequests !== false,
    logResponses: model.spec.observability?.logging?.logResponses || false,
    metricsEnabled: model.spec.observability?.metrics !== false,
    tracingEnabled: model.spec.observability?.tracing || false,
    requestsPerMinute: model.spec.rateLimiting?.requestsPerMinute,
    tokensPerMinute: model.spec.rateLimiting?.tokensPerMinute,
    concurrentRequests: model.spec.rateLimiting?.concurrentRequests,
    regions: model.spec.regions || [],
    retryMaxAttempts: model.spec.retryPolicy?.maxAttempts || 3,
    retryInitialBackoff: model.spec.retryPolicy?.initialBackoff || '1s',
    retryMaxBackoff: model.spec.retryPolicy?.maxBackoff || '30s',
    retryBackoffMultiplier: model.spec.retryPolicy?.backoffMultiplier || 2,
    retryableStatusCodes: model.spec.retryPolicy?.retryableStatusCodes || [429, 500, 502, 503, 504],
    egress: model.spec.egress?.map((rule: any) => ({
      ...rule,
      ports: rule.ports?.map((port: any) => 
        typeof port === 'object' && port.port ? port.port : port
      )
    })) || [],
    
    // Note: We don't populate apiKey for security reasons
  } : undefined

  if (isLoadingModel) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-64 w-full" />
      </div>
    )
  }

  if (!model) {
    return (
      <div className="space-y-6">
        <div className="text-center py-6">
          <p className="text-gray-600">Model not found</p>
        </div>
      </div>
    )
  }

  return (
    <div>
      <ModelForm
        initialData={initialData}
        isLoading={updateModel.isPending}
        error={updateModel.error?.message}
        onSubmit={handleSubmit}
        onCancel={handleCancel}
        isEdit={true}
        clusterName={clusterName}
      />
    </div>
  )
}