'use client'

import { useState, useEffect } from 'react'
import { useRouter, useParams } from 'next/navigation'
import { AuthenticatedLayout } from '@/components/layout/authenticated-layout'
import { ToolForm, ToolFormData } from '@/components/forms/tool-form'
import { ResourceHeader } from '@/components/ui/resource-header'
import { useTools } from '@/hooks/use-tools'
import { Wrench } from 'lucide-react'

export default function EditClusterToolPage() {
  const router = useRouter()
  const params = useParams()
  const clusterName = params?.name as string
  const toolName = params?.toolName as string
  
  const { data: tools, isLoading: isLoadingTools } = useTools({ clusterName })
  const tool = tools?.data?.find((t: any) => t.metadata.name === toolName)
  
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState('')
  const [initialData, setInitialData] = useState<Partial<ToolFormData>>()

  useEffect(() => {
    if (tool) {
      setInitialData({
        name: tool.metadata.name,
        type: tool.spec.type || '',
        description: tool.spec.description || '',
        image: tool.spec.image || '',
        endpoint: tool.spec.endpoint || '',
        port: tool.spec.port || 3000,
        healthCheckPath: tool.spec.healthCheckPath || '/health',
        envVars: tool.spec.envVars || [],
        resources: {
          cpu: tool.spec.resources?.requests?.cpu || '100m',
          memory: tool.spec.resources?.requests?.memory || '128Mi',
          cpuLimit: tool.spec.resources?.limits?.cpu || '500m',
          memoryLimit: tool.spec.resources?.limits?.memory || '512Mi'
        },
        enabled: tool.spec.enabled !== false,
        requireApproval: tool.spec.requireApproval || false,
        timeout: tool.spec.timeout || 30,
        retries: tool.spec.retries || 3,
      })
    }
  }, [tool])

  const handleSubmit = async (formData: ToolFormData) => {
    setIsLoading(true)
    setError('')

    try {
      const payload = {
        name: formData.name,
        type: formData.type,
        description: formData.description,
        image: formData.image,
        endpoint: formData.endpoint,
        port: formData.port,
        healthCheckPath: formData.healthCheckPath,
        envVars: formData.envVars,
        resources: formData.resources,
        enabled: formData.enabled,
        requireApproval: formData.requireApproval,
        timeout: formData.timeout,
        retries: formData.retries,
      }
      
      console.log('Updating tool with payload:', payload)
      
      const response = await fetch(`/api/tools/${toolName}`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(payload),
      })

      if (!response.ok) {
        const errorData = await response.json()
        console.error('API Error Response:', errorData)
        
        if (errorData.details && Array.isArray(errorData.details)) {
          const detailMessages = errorData.details.map((d: any) => `${d.path}: ${d.message}`).join(', ')
          throw new Error(`${errorData.error || 'Validation failed'}: ${detailMessages}`)
        }
        
        throw new Error(errorData.error || 'Failed to update tool')
      }

      const result = await response.json()
      console.log('Update tool result:', result)
      
      // Redirect to tool detail page
      router.push(`/clusters/${clusterName}/tools/${toolName}`)
    } catch (err: any) {
      console.error('Error updating tool:', err)
      setError(err.message || 'Failed to update tool')
    } finally {
      setIsLoading(false)
    }
  }

  const handleCancel = () => {
    router.push(`/clusters/${clusterName}/tools/${toolName}`)
  }

  if (isLoadingTools) {
    return (
      <AuthenticatedLayout>
        <div className="space-y-6">
          <div>
            <div className="h-8 w-48 bg-stone-200 dark:bg-stone-800 rounded animate-pulse"></div>
            <div className="h-4 w-64 bg-stone-200 dark:bg-stone-800 rounded mt-2 animate-pulse"></div>
          </div>
          <div className="h-96 bg-stone-200 dark:bg-stone-800 rounded animate-pulse"></div>
        </div>
      </AuthenticatedLayout>
    )
  }

  if (!tool) {
    return (
      <AuthenticatedLayout>
        <div className="space-y-6">
          <div>
            <h1 className="text-3xl font-bold">Tool Not Found</h1>
            <p className="text-muted-foreground mt-1">
              The tool "{toolName}" was not found in cluster "{clusterName}"
            </p>
          </div>
        </div>
      </AuthenticatedLayout>
    )
  }

  return (
    <AuthenticatedLayout>
      <div className="space-y-6">
        {/* Header */}
        <ResourceHeader
          backHref={`/clusters/${clusterName}/tools/${toolName}`}
          backLabel="Back to Tool"
          icon={Wrench}
          title={`Edit ${toolName}`}
          subtitle="Language Tool"
        />

        {/* Form */}
        <div className="max-w-4xl">
          {initialData ? (
            <ToolForm
              initialData={initialData}
              isLoading={isLoading}
              error={error}
              onSubmit={handleSubmit}
              onCancel={handleCancel}
              isEdit={true}
            />
          ) : (
            <div className="h-96 bg-stone-200 dark:bg-stone-800 rounded animate-pulse"></div>
          )}
        </div>
      </div>
    </AuthenticatedLayout>
  )
}