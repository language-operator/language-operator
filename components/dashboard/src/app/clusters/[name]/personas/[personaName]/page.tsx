'use client'

import { useRouter, useParams } from 'next/navigation'
import { AuthenticatedLayout } from '@/components/layout/authenticated-layout'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { ArrowLeft, Edit, Trash2, MessageSquare } from 'lucide-react'
import { usePersonas } from '@/hooks/use-personas'

export default function ClusterPersonaDetailPage() {
  const router = useRouter()
  const params = useParams()
  const clusterName = params?.name as string
  const personaName = params?.personaName as string

  const { data: personas, isLoading } = usePersonas()
  const persona = personas?.find((p: any) => p.metadata.name === personaName)

  const handleEdit = () => {
    router.push(`/clusters/${clusterName}/personas/${personaName}/edit`)
  }

  const handleDelete = async () => {
    if (!confirm(`Are you sure you want to delete the persona "${personaName}"?`)) {
      return
    }

    try {
      const response = await fetch(`/api/personas/${personaName}`, {
        method: 'DELETE',
      })

      if (!response.ok) {
        throw new Error('Failed to delete persona')
      }

      router.push(`/clusters/${clusterName}/personas`)
    } catch (error) {
      console.error('Error deleting persona:', error)
      alert('Failed to delete persona')
    }
  }

  const handleBack = () => {
    router.push(`/clusters/${clusterName}/personas`)
  }

  if (isLoading) {
    return (
      <AuthenticatedLayout>
        <div className="space-y-6">
          <div className="flex items-center space-x-4">
            <Button variant="outline" size="icon" onClick={handleBack}>
              <ArrowLeft className="h-4 w-4" />
            </Button>
            <div>
              <div className="h-8 w-48 bg-gray-200 rounded animate-pulse"></div>
              <div className="h-4 w-32 bg-gray-200 rounded mt-2 animate-pulse"></div>
            </div>
          </div>
        </div>
      </AuthenticatedLayout>
    )
  }

  if (!persona) {
    return (
      <AuthenticatedLayout>
        <div className="space-y-6">
          <div className="flex items-center space-x-4">
            <Button variant="outline" size="icon" onClick={handleBack}>
              <ArrowLeft className="h-4 w-4" />
            </Button>
            <div>
              <h1 className="text-3xl font-bold">Persona Not Found</h1>
              <p className="text-muted-foreground mt-1">
                The persona "{personaName}" was not found in cluster "{clusterName}"
              </p>
            </div>
          </div>
        </div>
      </AuthenticatedLayout>
    )
  }

  return (
    <AuthenticatedLayout>
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div className="flex items-center space-x-4">
            <Button variant="outline" size="icon" onClick={handleBack}>
              <ArrowLeft className="h-4 w-4" />
            </Button>
            <div>
              <h1 className="text-3xl font-bold">{persona.metadata.name}</h1>
              <p className="text-muted-foreground mt-1">
                Language Persona in {clusterName} cluster
              </p>
            </div>
          </div>
          
          <div className="flex items-center space-x-2">
            <Button variant="outline" onClick={handleEdit}>
              <Edit className="h-4 w-4 mr-2" />
              Edit
            </Button>
            <Button variant="destructive" onClick={handleDelete}>
              <Trash2 className="h-4 w-4 mr-2" />
              Delete
            </Button>
          </div>
        </div>

        {/* Persona Details */}
        <div className="grid gap-6">
          {/* Overview */}
          <Card>
            <CardHeader>
              <CardTitle>Overview</CardTitle>
              <CardDescription>Basic persona information and status</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <p className="text-sm font-medium">Name</p>
                  <p className="text-sm text-muted-foreground">{persona.metadata.name}</p>
                </div>
                <div>
                  <p className="text-sm font-medium">Role</p>
                  <p className="text-sm text-muted-foreground">{persona.spec.role || 'Unknown'}</p>
                </div>
                <div>
                  <p className="text-sm font-medium">Status</p>
                  <Badge variant={persona.status?.phase === 'Ready' ? 'default' : 'secondary'}>
                    {persona.status?.phase || 'Unknown'}
                  </Badge>
                </div>
                <div>
                  <p className="text-sm font-medium">Created</p>
                  <p className="text-sm text-muted-foreground">
                    {persona.metadata.creationTimestamp ? new Date(persona.metadata.creationTimestamp).toLocaleDateString() : 'Unknown'}
                  </p>
                </div>
              </div>

              {persona.spec.description && (
                <div>
                  <p className="text-sm font-medium">Description</p>
                  <p className="text-sm text-muted-foreground">{persona.spec.description}</p>
                </div>
              )}
            </CardContent>
          </Card>

          {/* Personality & Behavior */}
          <Card>
            <CardHeader>
              <CardTitle>Personality & Behavior</CardTitle>
              <CardDescription>System prompt and personality configuration</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              {persona.spec.systemPrompt && (
                <div>
                  <p className="text-sm font-medium">System Prompt</p>
                  <div className="text-sm text-muted-foreground bg-muted p-3 rounded-lg font-mono whitespace-pre-wrap">
                    {persona.spec.systemPrompt}
                  </div>
                </div>
              )}

              {persona.spec.traits && persona.spec.traits.length > 0 && (
                <div>
                  <p className="text-sm font-medium">Personality Traits</p>
                  <div className="flex flex-wrap gap-2 mt-2">
                    {persona.spec.traits.map((trait: any) => (
                      <Badge key={trait} variant="outline">
                        {trait}
                      </Badge>
                    ))}
                  </div>
                </div>
              )}

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <p className="text-sm font-medium">Temperature</p>
                  <p className="text-sm text-muted-foreground">{persona.spec.temperature || 0.7}</p>
                </div>
                <div>
                  <p className="text-sm font-medium">Max Tokens</p>
                  <p className="text-sm text-muted-foreground">{persona.spec.maxTokens || 2048}</p>
                </div>
              </div>
            </CardContent>
          </Card>

          {/* Examples */}
          {persona.spec.examples && persona.spec.examples.length > 0 && (
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center space-x-2">
                  <MessageSquare className="h-5 w-5" />
                  <span>Response Examples</span>
                </CardTitle>
                <CardDescription>Sample interactions demonstrating the persona's behavior</CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                {persona.spec.examples.map((example: any, index: number) => (
                  <div key={index} className="border rounded-lg p-4 space-y-2">
                    <div>
                      <p className="text-sm font-medium">User Input:</p>
                      <p className="text-sm text-muted-foreground bg-muted p-2 rounded">
                        {example.input}
                      </p>
                    </div>
                    <div>
                      <p className="text-sm font-medium">Expected Response:</p>
                      <p className="text-sm text-muted-foreground bg-muted p-2 rounded">
                        {example.output}
                      </p>
                    </div>
                  </div>
                ))}
              </CardContent>
            </Card>
          )}
        </div>
      </div>
    </AuthenticatedLayout>
  )
}