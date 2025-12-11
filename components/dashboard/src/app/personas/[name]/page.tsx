'use client'

import { useState } from 'react'
import { useParams } from 'next/navigation'
import Link from 'next/link'
import { AuthenticatedLayout } from '@/components/layout/authenticated-layout'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { 
  Users, AlertCircle, CheckCircle, ArrowLeft, Edit, Trash2, 
  User, Heart, MessageSquare, Bot, FileText, Star, Target
} from 'lucide-react'
import { usePersona, useDeletePersona } from '@/hooks/use-personas'
import { useAgents } from '@/hooks/use-agents'
import { LanguagePersona } from '@/types/persona'
import { Skeleton } from '@/components/ui/skeleton'

function formatTimeAgo(timestamp?: string) {
  if (!timestamp) return 'Unknown'
  const date = new Date(timestamp)
  const now = new Date()
  const diff = now.getTime() - date.getTime()
  const minutes = Math.floor(diff / 60000)
  const hours = Math.floor(diff / 3600000)
  const days = Math.floor(diff / 86400000)
  
  if (days > 0) return `${days} day${days !== 1 ? 's' : ''} ago`
  if (hours > 0) return `${hours} hour${hours !== 1 ? 's' : ''} ago`
  if (minutes > 0) return `${minutes} minute${minutes !== 1 ? 's' : ''} ago`
  return 'Just now'
}

function getToneColor(tone?: string) {
  if (!tone) return 'bg-gray-100 text-gray-800'
  
  const lowerTone = tone.toLowerCase()
  if (lowerTone.includes('professional') || lowerTone.includes('business')) {
    return 'bg-blue-100 text-blue-800'
  } else if (lowerTone.includes('friendly') || lowerTone.includes('warm')) {
    return 'bg-pink-100 text-pink-800'
  } else if (lowerTone.includes('technical') || lowerTone.includes('expert')) {
    return 'bg-purple-100 text-purple-800'
  } else if (lowerTone.includes('creative') || lowerTone.includes('inspiring')) {
    return 'bg-orange-100 text-orange-800'
  } else {
    return 'bg-gray-100 text-gray-800'
  }
}

interface PersonaOverviewProps {
  persona: LanguagePersona
}

function PersonaOverview({ persona }: PersonaOverviewProps) {
  return (
    <div className="space-y-6">
      {/* Basic Info */}
      <div className="grid gap-6 md:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>Basic Information</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div>
              <p className="text-sm font-medium text-muted-foreground">Name</p>
              <p className="text-sm">{persona.metadata.name}</p>
            </div>
            <div>
              <p className="text-sm font-medium text-muted-foreground">Display Name</p>
              <p className="text-sm">{persona.spec.displayName || persona.metadata.name}</p>
            </div>
            <div>
              <p className="text-sm font-medium text-muted-foreground">Namespace</p>
              <p className="text-sm">{persona.metadata.namespace}</p>
            </div>
            <div>
              <p className="text-sm font-medium text-muted-foreground">Description</p>
              <p className="text-sm">{persona.spec.description || 'No description provided'}</p>
            </div>
            <div>
              <p className="text-sm font-medium text-muted-foreground">Created</p>
              <p className="text-sm">{formatTimeAgo(persona.metadata.creationTimestamp)}</p>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Personality Configuration</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div>
              <p className="text-sm font-medium text-muted-foreground">Tone</p>
              <div className="flex items-center space-x-2">
                <Heart className="h-4 w-4 text-pink-500" />
                <Badge className={getToneColor(persona.spec.tone)}>
                  {persona.spec.tone || 'Not specified'}
                </Badge>
              </div>
            </div>
            {persona.spec.personality && (
              <div>
                <p className="text-sm font-medium text-muted-foreground">Personality Traits</p>
                <div className="flex flex-wrap gap-1 mt-1">
                  {persona.spec.personality.map((trait, index) => (
                    <Badge key={index} variant="outline" className="text-xs">
                      {trait}
                    </Badge>
                  ))}
                </div>
              </div>
            )}
            {persona.spec.goals && persona.spec.goals.length > 0 && (
              <div>
                <p className="text-sm font-medium text-muted-foreground">Goals</p>
                <div className="space-y-1 mt-1">
                  {persona.spec.goals.map((goal, index) => (
                    <div key={index} className="flex items-start space-x-2">
                      <Target className="h-3 w-3 text-blue-500 mt-0.5" />
                      <span className="text-xs">{goal}</span>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Behavior Instructions */}
      {persona.spec.instructions && (
        <Card>
          <CardHeader>
            <CardTitle>Behavior Instructions</CardTitle>
            <CardDescription>Core instructions that define how this persona should behave</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="bg-gray-50 p-4 rounded-lg">
              <pre className="text-sm whitespace-pre-wrap font-mono">
                {persona.spec.instructions}
              </pre>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Examples */}
      {persona.spec.examples && persona.spec.examples.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle>Example Prompts & Responses</CardTitle>
            <CardDescription>Sample interactions that demonstrate this persona's style</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              {persona.spec.examples.map((example, index) => (
                <div key={index} className="border rounded-lg p-4">
                  <div className="space-y-3">
                    <div>
                      <p className="text-xs font-medium text-muted-foreground uppercase tracking-wide">User Input</p>
                      <div className="bg-blue-50 p-3 rounded mt-1">
                        <p className="text-sm">{example.input}</p>
                      </div>
                    </div>
                    <div>
                      <p className="text-xs font-medium text-muted-foreground uppercase tracking-wide">Expected Response</p>
                      <div className="bg-green-50 p-3 rounded mt-1">
                        <p className="text-sm">{example.output}</p>
                      </div>
                    </div>
                    {example.context && (
                      <div>
                        <p className="text-xs font-medium text-muted-foreground uppercase tracking-wide">Context</p>
                        <p className="text-xs text-muted-foreground mt-1">{example.context}</p>
                      </div>
                    )}
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      )}

      {/* Constraints */}
      {persona.spec.constraints && persona.spec.constraints.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle>Behavioral Constraints</CardTitle>
            <CardDescription>Rules and limitations for this persona</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-2">
              {persona.spec.constraints.map((constraint, index) => (
                <div key={index} className="flex items-start space-x-2 p-2 bg-yellow-50 rounded">
                  <AlertCircle className="h-4 w-4 text-yellow-600 mt-0.5" />
                  <span className="text-sm">{constraint}</span>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      )}

      {/* Validation Results */}
      {persona.status?.validation && (
        <Card>
          <CardHeader>
            <CardTitle>Validation Status</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div>
              <p className="text-sm font-medium text-muted-foreground">Overall Status</p>
              <div className="flex items-center space-x-2">
                {persona.status.validation.valid ? (
                  <>
                    <CheckCircle className="h-4 w-4 text-green-500" />
                    <Badge variant="default" className="bg-green-100 text-green-800">Valid</Badge>
                  </>
                ) : (
                  <>
                    <AlertCircle className="h-4 w-4 text-red-500" />
                    <Badge variant="destructive">Invalid</Badge>
                  </>
                )}
              </div>
            </div>
            {persona.status.validation.errors && persona.status.validation.errors.length > 0 && (
              <div>
                <p className="text-sm font-medium text-muted-foreground">Validation Errors</p>
                <div className="space-y-1 mt-1">
                  {persona.status.validation.errors.map((error, index) => (
                    <div key={index} className="text-sm text-red-600 bg-red-50 p-2 rounded">
                      {error}
                    </div>
                  ))}
                </div>
              </div>
            )}
            <div>
              <p className="text-sm font-medium text-muted-foreground">Last Validated</p>
              <p className="text-sm">{formatTimeAgo(persona.status.validation.lastValidated)}</p>
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  )
}

interface PersonaUsageProps {
  persona: LanguagePersona
}

function PersonaUsage({ persona }: PersonaUsageProps) {
  // Fetch agents to see which ones use this persona
  const { data: agentsResponse } = useAgents({
    limit: 100,
  })

  const allAgents = agentsResponse?.data || []
  const usingAgents = allAgents.filter(agent => 
    agent.spec.persona?.name === persona.metadata.name
  )

  return (
    <div className="space-y-6">
      {/* Usage Summary */}
      <div className="grid gap-4 md:grid-cols-3">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Using Agents</CardTitle>
            <Bot className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{usingAgents.length}</div>
            <p className="text-xs text-muted-foreground">
              Active agents
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Usage Metrics</CardTitle>
            <Star className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {persona.status?.metrics?.usageCount?.toLocaleString() || 'N/A'}
            </div>
            <p className="text-xs text-muted-foreground">
              Total invocations
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Success Rate</CardTitle>
            <CheckCircle className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {persona.status?.metrics?.successRate || 'N/A'}
            </div>
            <p className="text-xs text-muted-foreground">
              Success percentage
            </p>
          </CardContent>
        </Card>
      </div>

      {/* Agents Using This Persona */}
      <Card>
        <CardHeader>
          <CardTitle>Agents Using This Persona</CardTitle>
          <CardDescription>Language agents that have been configured with this persona</CardDescription>
        </CardHeader>
        <CardContent>
          {usingAgents.length > 0 ? (
            <div className="space-y-3">
              {usingAgents.map((agent) => (
                <div key={agent.metadata.name} className="flex items-center justify-between p-3 border rounded-lg">
                  <div className="flex items-center space-x-3">
                    <Bot className="h-5 w-5 text-blue-500" />
                    <div>
                      <p className="text-sm font-medium">{agent.metadata.name}</p>
                      <p className="text-xs text-muted-foreground">
                        {agent.spec.executionMode} mode • {agent.metadata.namespace}
                      </p>
                    </div>
                  </div>
                  <div className="flex items-center space-x-2">
                    <Badge variant={agent.status?.phase === 'Running' ? 'default' : 'secondary'}>
                      {agent.status?.phase || 'Unknown'}
                    </Badge>
                    <Link href={`/agents/${agent.metadata.name}`}>
                      <Button variant="outline" size="sm">
                        View Agent
                      </Button>
                    </Link>
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <div className="text-center py-8">
              <Bot className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
              <h3 className="text-lg font-medium mb-2">No agents using this persona</h3>
              <p className="text-muted-foreground mb-4">
                This persona hasn't been assigned to any agents yet.
              </p>
              <Link href="/agents/new">
                <Button>
                  Create Agent with This Persona
                </Button>
              </Link>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Recent Activity */}
      {persona.status?.metrics?.recentActivity && (
        <Card>
          <CardHeader>
            <CardTitle>Recent Activity</CardTitle>
            <CardDescription>Latest usage statistics for this persona</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              <div className="grid gap-4 md:grid-cols-2">
                <div>
                  <p className="text-sm font-medium text-muted-foreground">Last Used</p>
                  <p className="text-sm">{formatTimeAgo(persona.status.metrics.lastUsed)}</p>
                </div>
                <div>
                  <p className="text-sm font-medium text-muted-foreground">Average Response Quality</p>
                  <p className="text-sm">{persona.status.metrics.averageQuality || 'N/A'}</p>
                </div>
              </div>
              {persona.status.metrics.recentFeedback && (
                <div>
                  <p className="text-sm font-medium text-muted-foreground">Recent Feedback</p>
                  <div className="space-y-2 mt-2">
                    {persona.status.metrics.recentFeedback.map((feedback, index) => (
                      <div key={index} className="p-2 bg-gray-50 rounded text-sm">
                        <div className="flex items-center justify-between">
                          <span className="font-medium">Rating: {feedback.rating}/5</span>
                          <span className="text-xs text-muted-foreground">{formatTimeAgo(feedback.timestamp)}</span>
                        </div>
                        {feedback.comment && (
                          <p className="text-xs mt-1 text-muted-foreground">{feedback.comment}</p>
                        )}
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  )
}

export default function PersonaDetailPage() {
  const params = useParams()
  const personaName = params.name as string
  const [activeTab, setActiveTab] = useState('overview')
  
  const { data: personaResponse, isLoading, error } = usePersona(personaName)
  const deletePersona = useDeletePersona()

  const persona = personaResponse?.data

  const handleDeletePersona = async () => {
    if (!persona || !persona.metadata.name) return
    
    if (confirm(`Are you sure you want to delete persona "${persona.metadata.name}"?`)) {
      try {
        await deletePersona.mutateAsync(persona.metadata.name)
        // Redirect to personas list after successful deletion
        window.location.href = '/personas'
      } catch (error) {
        console.error('Failed to delete persona:', error)
        alert('Failed to delete persona. Please try again.')
      }
    }
  }

  if (isLoading) {
    return (
      <AuthenticatedLayout>
        <div className="space-y-6">
          <div className="flex items-center space-x-4">
            <Skeleton className="h-8 w-8" />
            <div className="space-y-2">
              <Skeleton className="h-6 w-48" />
              <Skeleton className="h-4 w-32" />
            </div>
          </div>
          <Skeleton className="h-64 w-full" />
        </div>
      </AuthenticatedLayout>
    )
  }

  if (error || !persona) {
    return (
      <AuthenticatedLayout>
        <div className="flex items-center justify-center h-64">
          <div className="text-center">
            <AlertCircle className="h-12 w-12 text-red-500 mx-auto mb-4" />
            <h3 className="text-lg font-medium mb-2">Persona not found</h3>
            <p className="text-muted-foreground mb-4">
              The persona "{personaName}" could not be found.
            </p>
            <Link href="/personas">
              <Button variant="outline">
                <ArrowLeft className="h-4 w-4 mr-2" />
                Back to Personas
              </Button>
            </Link>
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
            <Link href="/personas">
              <Button variant="outline" size="icon">
                <ArrowLeft className="h-4 w-4" />
              </Button>
            </Link>
            <Users className="h-8 w-8 text-purple-500" />
            <div>
              <div className="flex items-center space-x-3">
                <h1 className="text-3xl font-bold">{persona.spec.displayName || persona.metadata.name}</h1>
                <div className="flex items-center space-x-2">
                  <Heart className="h-5 w-5 text-pink-500" />
                  <Badge className={getToneColor(persona.spec.tone)}>
                    {persona.spec.tone || 'No tone specified'}
                  </Badge>
                </div>
              </div>
              <p className="text-muted-foreground">
                Language Persona • {persona.metadata.namespace}
              </p>
            </div>
          </div>
          
          <div className="flex items-center space-x-2">
            <Link href={`/personas/${persona.metadata.name}/edit`}>
              <Button variant="outline">
                <Edit className="h-4 w-4 mr-2" />
                Edit
              </Button>
            </Link>
            <Button 
              variant="destructive"
              onClick={handleDeletePersona}
              disabled={deletePersona.isPending}
            >
              <Trash2 className="h-4 w-4 mr-2" />
              {deletePersona.isPending ? 'Deleting...' : 'Delete'}
            </Button>
          </div>
        </div>

        {/* Tabs */}
        <Tabs value={activeTab} onValueChange={setActiveTab} className="space-y-6">
          <TabsList>
            <TabsTrigger value="overview">Overview</TabsTrigger>
            <TabsTrigger value="usage">Usage by Agents</TabsTrigger>
          </TabsList>

          <TabsContent value="overview" className="space-y-6">
            <PersonaOverview persona={persona} />
          </TabsContent>

          <TabsContent value="usage" className="space-y-6">
            <PersonaUsage persona={persona} />
          </TabsContent>
        </Tabs>
      </div>
    </AuthenticatedLayout>
  )
}