'use client'

import { useState, useEffect } from 'react'
import { useRouter } from 'next/navigation'
import { AuthenticatedLayout } from '@/components/layout/authenticated-layout'
import { Button } from '@/components/ui/button'
import { ToolForm, ToolFormData } from '@/components/forms/tool-form'
import { ArrowLeft } from 'lucide-react'
import { Skeleton } from '@/components/ui/skeleton'
import Link from 'next/link'

interface Tool {
  metadata: {
    name: string
    namespace: string
    creationTimestamp: string
  }
  spec: {
    type: string
    image?: string
    endpoint?: string
    port?: number
    healthCheckPath?: string
    environment?: Record<string, string>
    resources?: {
      requests?: {
        cpu?: string
        memory?: string
      }
      limits?: {
        cpu?: string
        memory?: string
      }
    }
    enabled?: boolean
    requireApproval?: boolean
    timeout?: number
    retries?: number
  }
  status: {
    phase: string
  }
}

export default function EditToolPage({ params }: { params: Promise<{ name: string }> }) {
  const router = useRouter()
  const [tool, setTool] = useState<Tool | null>(null)
  const [isLoading, setIsLoading] = useState(false)
  const [isLoadingTool, setIsLoadingTool] = useState(true)
  const [error, setError] = useState('')
  const [toolName, setToolName] = useState<string>('')

  // Get the tool name from params
  useEffect(() => {
    const getToolName = async () => {
      const resolvedParams = await params
      setToolName(resolvedParams.name)
    }
    getToolName()
  }, [params])

  // Fetch existing tool data
  useEffect(() => {
    if (!toolName) return

    const fetchTool = async () => {
      setIsLoadingTool(true)
      try {
        const response = await fetch(`/api/tools/${toolName}`)
        if (!response.ok) {
          throw new Error('Failed to fetch tool')
        }
        const data = await response.json()
        setTool(data.tool)
      } catch (err: any) {
        console.error('Error fetching tool:', err)
        setError(err.message || 'Failed to load tool')
      } finally {
        setIsLoadingTool(false)
      }
    }

    fetchTool()
  }, [toolName])

  const handleSubmit = async (formData: ToolFormData) => {
    setIsLoading(true)
    setError('')

    try {
      const response = await fetch(`/api/tools/${toolName}`, {
        method: 'PATCH',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          type: formData.type,
          description: formData.description || undefined,
          spec: {
            type: formData.type,
            ...(formData.image && { image: formData.image }),
            ...(formData.endpoint && { endpoint: formData.endpoint }),
            ...(formData.port && { port: formData.port }),
            ...(formData.healthCheckPath && { healthCheckPath: formData.healthCheckPath }),
            environment: formData.envVars.filter(env => env.key && env.value).reduce((acc, env) => {
              acc[env.key] = env.value
              return acc
            }, {} as Record<string, string>),
            resources: {
              requests: {
                cpu: formData.resources.cpu,
                memory: formData.resources.memory
              },
              limits: {
                cpu: formData.resources.cpuLimit,
                memory: formData.resources.memoryLimit
              }
            },
            enabled: formData.enabled,
            requireApproval: formData.requireApproval,
            timeout: formData.timeout,
            retries: formData.retries,
          },
        }),
      })

      if (!response.ok) {
        const errorData = await response.json()
        throw new Error(errorData.error || 'Failed to update tool')
      }

      // Redirect to tool details page
      router.push(`/tools/${toolName}`)
    } catch (err: any) {
      console.error('Error updating tool:', err)
      setError(err.message || 'Failed to update tool')
    } finally {
      setIsLoading(false)
    }
  }

  const handleCancel = () => {
    router.push(`/tools/${toolName}`)
  }

  // Convert tool data to form format
  const getInitialFormData = (): Partial<ToolFormData> | undefined => {
    if (!tool) return undefined

    // Convert environment object to array format
    const envVars = tool.spec.environment ? 
      Object.entries(tool.spec.environment).map(([key, value]) => ({ key, value })) : []

    return {
      name: tool.metadata.name,
      type: tool.spec.type,
      description: '', // Tools don't have description in current schema
      image: tool.spec.image || '',
      endpoint: tool.spec.endpoint || '',
      port: tool.spec.port || 3000,
      healthCheckPath: tool.spec.healthCheckPath || '/health',
      envVars,
      resources: {
        cpu: tool.spec.resources?.requests?.cpu || '100m',
        memory: tool.spec.resources?.requests?.memory || '128Mi',
        cpuLimit: tool.spec.resources?.limits?.cpu || '500m',
        memoryLimit: tool.spec.resources?.limits?.memory || '512Mi',
      },
      enabled: tool.spec.enabled ?? true,
      requireApproval: tool.spec.requireApproval ?? false,
      timeout: tool.spec.timeout || 30,
      retries: tool.spec.retries || 3,
    }
  }

  if (isLoadingTool) {
    return (
      <AuthenticatedLayout>
        <div className="space-y-6">
          <div className="flex items-center space-x-4">
            <Skeleton className="h-8 w-32" />
            <div>
              <Skeleton className="h-8 w-64 mb-2" />
              <Skeleton className="h-4 w-48" />
            </div>
          </div>
          <div className="max-w-2xl space-y-6">
            <Skeleton className="h-48" />
            <Skeleton className="h-32" />
            <Skeleton className="h-48" />
            <Skeleton className="h-32" />
          </div>
        </div>
      </AuthenticatedLayout>
    )
  }

  if (error && !tool) {
    return (
      <AuthenticatedLayout>
        <div className="space-y-6">
          <div className="flex items-center space-x-4">
            <Link href="/tools">
              <Button variant="outline" size="sm">
                <ArrowLeft className="h-4 w-4 mr-2" />
                Back to Tools
              </Button>
            </Link>
            <div>
              <h1 className="text-3xl font-bold">Edit Tool</h1>
              <p className="text-muted-foreground">Failed to load tool</p>
            </div>
          </div>
          <div className="max-w-2xl">
            <div className="text-center py-12">
              <h3 className="text-lg font-medium mb-2">Error loading tool</h3>
              <p className="text-muted-foreground mb-4">{error}</p>
              <Link href="/tools">
                <Button>Back to Tools</Button>
              </Link>
            </div>
          </div>
        </div>
      </AuthenticatedLayout>
    )
  }

  return (
    <AuthenticatedLayout>
      <div className="space-y-6">
        <div className="flex items-center space-x-4">
          <Link href={`/tools/${toolName}`}>
            <Button variant="outline" size="sm">
              <ArrowLeft className="h-4 w-4 mr-2" />
              Back to Tool
            </Button>
          </Link>
          <div>
            <h1 className="text-3xl font-bold">Edit Tool</h1>
            <p className="text-muted-foreground">
              Update settings for tool "{toolName}"
            </p>
          </div>
        </div>

        <div className="max-w-2xl">
          <ToolForm
            initialData={getInitialFormData()}
            isLoading={isLoading}
            error={error}
            onSubmit={handleSubmit}
            onCancel={handleCancel}
            isEdit={true}
          />
        </div>
      </div>
    </AuthenticatedLayout>
  )
}