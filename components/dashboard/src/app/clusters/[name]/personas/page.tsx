'use client'

import { useParams } from 'next/navigation'
import { AuthenticatedLayout } from '@/components/layout/authenticated-layout'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Users, Plus, MessageCircle, Palette, Clock } from 'lucide-react'
import Link from 'next/link'
import { usePersonas } from '@/hooks/use-personas'

export default function ClusterPersonas() {
  const params = useParams()
  const clusterName = params?.name as string
  
  // Fetch all personas and filter client-side (TODO: implement server-side filtering by namespace)
  const { data: personasResponse, isLoading, error } = usePersonas({ limit: 100 })
  const allPersonas = personasResponse?.data || []
  
  // For now, show all personas (TODO: implement proper cluster-scoped filtering)
  const clusterPersonas = allPersonas

  const getStatusColor = (phase?: string) => {
    switch (phase) {
      case 'Ready':
      case 'Available':
        return 'bg-green-100 text-green-800'
      case 'Pending':
      case 'Validating':
        return 'bg-yellow-100 text-yellow-800'
      case 'Failed':
      case 'Error':
        return 'bg-red-100 text-red-800'
      default:
        return 'bg-gray-100 text-gray-800'
    }
  }

  const getToneColor = (tone?: string) => {
    switch (tone) {
      case 'professional':
        return 'bg-blue-100 text-blue-800'
      case 'friendly':
        return 'bg-green-100 text-green-800'
      case 'casual':
        return 'bg-orange-100 text-orange-800'
      case 'formal':
        return 'bg-purple-100 text-purple-800'
      case 'empathetic':
        return 'bg-pink-100 text-pink-800'
      default:
        return 'bg-gray-100 text-gray-800'
    }
  }

  return (
    <AuthenticatedLayout>
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold">Personas</h1>
            <p className="text-gray-600 mt-1">
              AI personas configured for the {clusterName} cluster
            </p>
          </div>
          <Button asChild>
            <Link href={`/clusters/${clusterName}/personas/new`}>
              <Plus className="h-4 w-4 mr-2" />
              New Persona
            </Link>
          </Button>
        </div>

        {/* Loading State */}
        {isLoading && (
          <Card>
            <CardContent className="flex items-center justify-center py-16">
              <div className="text-center">
                <Users className="h-8 w-8 animate-pulse mx-auto mb-4 text-gray-400" />
                <p className="text-gray-600">Loading personas...</p>
              </div>
            </CardContent>
          </Card>
        )}

        {/* Error State */}
        {error && (
          <Card>
            <CardContent className="flex items-center justify-center py-16">
              <div className="text-center">
                <Users className="h-8 w-8 mx-auto mb-4 text-red-400" />
                <p className="text-red-600 mb-2">Failed to load personas</p>
                <p className="text-gray-600 text-sm">{error.message}</p>
              </div>
            </CardContent>
          </Card>
        )}

        {/* Personas List */}
        {!isLoading && !error && (
          <>
            {clusterPersonas.length === 0 ? (
              /* Empty State */
              <Card>
                <CardContent className="flex flex-col items-center justify-center py-16">
                  <Users className="h-16 w-16 text-gray-400 mb-4" />
                  <CardTitle className="text-xl mb-2">No personas yet</CardTitle>
                  <CardDescription className="text-center max-w-md mb-6">
                    Personas define the behavior, knowledge, and communication style 
                    for AI agents. Create your first persona to get started.
                  </CardDescription>
                  <Button asChild>
                    <Link href={`/clusters/${clusterName}/personas/new`}>
                      <Plus className="h-4 w-4 mr-2" />
                      Create Your First Persona
                    </Link>
                  </Button>
                </CardContent>
              </Card>
            ) : (
              /* Personas Grid */
              <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
                {clusterPersonas.map((persona: any) => (
                  <Card key={persona.metadata.name} className="hover:shadow-md transition-shadow">
                    <CardHeader className="pb-3">
                      <div className="flex items-center justify-between">
                        <CardTitle className="text-lg">
                          {persona.spec.displayName || persona.metadata.name}
                        </CardTitle>
                        <Badge className={getStatusColor(persona.status?.phase)}>
                          {persona.status?.phase || 'Unknown'}
                        </Badge>
                      </div>
                      <CardDescription>
                        {persona.spec.description || 'No description provided'}
                      </CardDescription>
                    </CardHeader>
                    <CardContent>
                      <div className="space-y-3">
                        {persona.spec.tone && (
                          <div className="flex items-center space-x-2 text-sm">
                            <Palette className="h-4 w-4 text-gray-500" />
                            <Badge className={getToneColor(persona.spec.tone)} variant="secondary">
                              {persona.spec.tone}
                            </Badge>
                          </div>
                        )}
                        
                        {persona.spec.systemPrompt && (
                          <div className="space-y-1">
                            <div className="flex items-center space-x-2 text-sm text-gray-600">
                              <MessageCircle className="h-4 w-4" />
                              <span>System Prompt</span>
                            </div>
                            <p className="text-xs text-gray-600 line-clamp-2">
                              {persona.spec.systemPrompt}
                            </p>
                          </div>
                        )}

                        <div className="pt-2 border-t">
                          <div className="text-xs text-gray-500 flex items-center space-x-1">
                            <Clock className="h-3 w-3" />
                            <span>
                              Created: {new Date(persona.metadata.creationTimestamp).toLocaleDateString()}
                            </span>
                          </div>
                        </div>
                      </div>
                    </CardContent>
                  </Card>
                ))}
              </div>
            )}
          </>
        )}
      </div>
    </AuthenticatedLayout>
  )
}