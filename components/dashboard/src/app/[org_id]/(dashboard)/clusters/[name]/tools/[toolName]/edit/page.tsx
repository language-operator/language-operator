'use client'

import { useState, useEffect } from 'react'
import { useRouter, useParams } from 'next/navigation'
import { AuthenticatedLayout } from '@/components/layout/authenticated-layout'
import { ToolEditForm, ToolEditFormData } from '@/components/forms/tool-edit-form'
import { ResourceHeader } from '@/components/ui/resource-header'
import { useTool } from '@/hooks/use-tools'
import { useOrganization } from '@/components/organization-provider'
import { Wrench } from 'lucide-react'

export default function EditClusterToolPage() {
  const router = useRouter()
  const params = useParams()
  const { getOrgUrl } = useOrganization()
  const clusterName = params?.name as string
  const toolName = params?.toolName as string
  
  const { data: toolResponse, isLoading: isLoadingTool, error: fetchError } = useTool(toolName, clusterName)
  const tool = toolResponse?.data
  
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState('')
  const [initialData, setInitialData] = useState<Partial<ToolEditFormData>>()

  useEffect(() => {
    if (tool) {
      setInitialData({
        image: tool.spec.image || '',
        port: tool.spec.port || 3000,
        deploymentMode: (tool.spec.deploymentMode || 'service') as 'service' | 'sidecar',
        envVars: tool.spec.env?.map((env: { name: string; value?: string }) => ({
          key: env.name,
          value: env.value || ''
        })) || [],
        resources: {
          cpu: tool.spec.resources?.requests?.cpu || '100m',
          memory: tool.spec.resources?.requests?.memory || '128Mi',
          cpuLimit: tool.spec.resources?.limits?.cpu || '500m',
          memoryLimit: tool.spec.resources?.limits?.memory || '512Mi'
        }
      })
    }
  }, [tool])

  const handleSubmit = async (formData: ToolEditFormData) => {
    setIsLoading(true)
    setError('')

    try {
      console.log('Updating tool with payload:', formData)
      
      const response = await fetch(`/api/clusters/${clusterName}/tools/${toolName}`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(formData),
      })

      if (!response.ok) {
        const errorData = await response.json()
        console.error('API Error Response:', errorData)
        
        if (errorData.details && Array.isArray(errorData.details)) {
          const detailMessages = errorData.details.map((d: { path: string; message: string }) => `${d.path}: ${d.message}`).join(', ')
          throw new Error(`${errorData.error || 'Validation failed'}: ${detailMessages}`)
        }
        
        throw new Error(errorData.error || 'Failed to update tool')
      }

      const result = await response.json()
      console.log('Update tool result:', result)
      
      // Redirect to tool detail page
      router.push(getOrgUrl(`/clusters/${clusterName}/tools/${toolName}`))
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to update tool'
      console.error('Error updating tool:', err)
      setError(errorMessage)
    } finally {
      setIsLoading(false)
    }
  }

  const handleCancel = () => {
    router.push(getOrgUrl(`/clusters/${clusterName}/tools/${toolName}`))
  }

  if (isLoadingTool) {
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

  if (fetchError || (!isLoadingTool && !tool)) {
    return (
      <AuthenticatedLayout>
        <div className="space-y-6">
          <div>
            <h1 className="text-3xl font-bold">Tool Not Found</h1>
            <p className="text-muted-foreground mt-1">
              The tool &ldquo;{toolName}&rdquo; was not found in cluster &ldquo;{clusterName}&rdquo;
            </p>
            {fetchError && (
              <p className="text-sm text-red-600 mt-2">
                Error: {fetchError instanceof Error ? fetchError.message : 'Unknown error'}
              </p>
            )}
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
          title={toolName}
          subtitle="LanguageTool"
        />

        {/* Form */}
        <div>
          {initialData ? (
            <ToolEditForm
              initialData={initialData}
              isLoading={isLoading}
              error={error}
              onSubmit={handleSubmit}
              onCancel={handleCancel}
            />
          ) : (
            <div className="h-96 bg-stone-200 dark:bg-stone-800 rounded animate-pulse"></div>
          )}
        </div>
      </div>
    </AuthenticatedLayout>
  )
}