'use client'

import { useState } from 'react'
import { useRouter } from 'next/navigation'
import { AuthenticatedLayout } from '@/components/layout/authenticated-layout'
import { Button } from '@/components/ui/button'
import { ToolForm, ToolFormData } from '@/components/forms/tool-form'
import { ArrowLeft } from 'lucide-react'
import Link from 'next/link'

export default function CreateToolPage() {
  const router = useRouter()
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState('')

  const handleSubmit = async (formData: ToolFormData) => {
    setIsLoading(true)
    setError('')

    try {
      const response = await fetch('/api/tools', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          name: formData.name,
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
        throw new Error(errorData.error || 'Failed to create tool')
      }

      const result = await response.json()
      
      // Redirect to tool details page
      router.push(`/tools/${result.tool.metadata.name}`)
    } catch (err: any) {
      console.error('Error creating tool:', err)
      setError(err.message || 'Failed to create tool')
    } finally {
      setIsLoading(false)
    }
  }

  const handleCancel = () => {
    router.push('/tools')
  }

  return (
    <AuthenticatedLayout>
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center space-x-4">
          <Link href="/tools">
            <Button variant="outline" size="sm">
              <ArrowLeft className="h-4 w-4 mr-2" />
              Back to Tools
            </Button>
          </Link>
          <div>
            <h1 className="text-3xl font-bold">Create Language Tool</h1>
            <p className="text-muted-foreground">
              Add a new tool to extend agent capabilities
            </p>
          </div>
        </div>

        {/* Form */}
        <div className="max-w-2xl">
          <ToolForm
            isLoading={isLoading}
            error={error}
            onSubmit={handleSubmit}
            onCancel={handleCancel}
          />
        </div>
      </div>
    </AuthenticatedLayout>
  )
}