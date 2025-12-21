'use client'

import { useState } from 'react'
import { useRouter, useParams } from 'next/navigation'
import { fetchWithOrganization } from '@/lib/api-client'
import { AuthenticatedLayout } from '@/components/layout/authenticated-layout'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { 
  Users, AlertCircle, CheckCircle, Clock, ArrowLeft, 
  Edit, Trash2, MessageCircle, Palette, BookOpen, Target, MoreVertical, FileCode, Copy, Check
} from 'lucide-react'
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter'
import { oneLight } from 'react-syntax-highlighter/dist/esm/styles/prism'
import { usePersona, useDeletePersona } from '@/hooks/use-personas'
import { Skeleton } from '@/components/ui/skeleton'
import { ResourceHeader } from '@/components/ui/resource-header'
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from '@/components/ui/dropdown-menu'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { ResourceEventsActivity } from '@/components/ui/events-activity'

function formatTimeAgo(timestamp?: string | Date) {
  if (!timestamp) return 'Unknown'
  const date = typeof timestamp === 'string' ? new Date(timestamp) : timestamp
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

export default function ClusterPersonaDetailPage() {
  const router = useRouter()
  const params = useParams()
  const clusterName = params?.name as string
  const personaName = params?.personaName as string
  const [activeTab, setActiveTab] = useState('overview')
  const [yamlModalOpen, setYamlModalOpen] = useState(false)
  const [yamlContent, setYamlContent] = useState('')
  const [yamlLoading, setYamlLoading] = useState(false)
  const [copied, setCopied] = useState(false)
  
  const { data: personaResponse, isLoading, error } = usePersona(personaName, clusterName)
  const deletePersona = useDeletePersona(clusterName)

  const persona = personaResponse?.persona

  const getStatusIcon = (persona: any) => {
    const phase = persona?.status?.phase || 'Unknown'
    
    if (phase === 'Ready' || phase === 'Available') {
      return <CheckCircle className="h-5 w-5 text-green-500" />
    } else if (phase === 'Pending' || phase === 'Validating') {
      return <Clock className="h-5 w-5 text-yellow-500" />
    } else if (phase === 'Failed' || phase === 'Error') {
      return <AlertCircle className="h-5 w-5 text-red-500" />
    } else {
      return <AlertCircle className="h-5 w-5 text-gray-500" />
    }
  }

  const getStatusColor = (persona: any) => {
    const phase = persona?.status?.phase || 'Unknown'
    
    if (phase === 'Ready' || phase === 'Available') {
      return 'bg-green-100 text-green-800'
    } else if (phase === 'Pending' || phase === 'Validating') {
      return 'bg-yellow-100 text-yellow-800'
    } else if (phase === 'Failed' || phase === 'Error') {
      return 'bg-red-100 text-red-800'
    } else {
      return 'bg-gray-100 text-gray-800'
    }
  }

  const handleDeletePersona = async () => {
    if (!persona || !persona.metadata.name) return
    
    if (confirm(`Are you sure you want to delete persona "${persona.spec.displayName || persona.metadata.name}"?`)) {
      try {
        await deletePersona.mutateAsync(persona.metadata.name)
        router.push(`/clusters/${clusterName}/personas`)
      } catch (error) {
        console.error('Failed to delete persona:', error)
        alert('Failed to delete persona. Please try again.')
      }
    }
  }

  const handleBack = () => {
    router.push(`/clusters/${clusterName}/personas`)
  }

  const handleViewYaml = async () => {
    setYamlModalOpen(true)
    setYamlLoading(true)
    try {
      const response = await fetchWithOrganization(`/api/clusters/${clusterName}/personas/${personaName}/yaml`)
      if (!response.ok) {
        throw new Error('Failed to fetch YAML')
      }
      const yaml = await response.text()
      setYamlContent(yaml)
    } catch (error) {
      console.error('Error fetching YAML:', error)
      setYamlContent('Error loading YAML content')
    } finally {
      setYamlLoading(false)
    }
  }

  const handleCopyYaml = async () => {
    try {
      await navigator.clipboard.writeText(yamlContent)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    } catch (error) {
      console.error('Failed to copy YAML:', error)
    }
  }

  if (isLoading) {
    return (
      <AuthenticatedLayout>
        <div className="space-y-6">
          <div className="flex items-center space-x-4">
            <Button variant="outline" size="icon" onClick={handleBack}>
              <ArrowLeft className="h-4 w-4" />
            </Button>
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
              The persona "{personaName}" could not be found in cluster "{clusterName}".
            </p>
            <Button variant="outline" onClick={handleBack}>
              <ArrowLeft className="h-4 w-4 mr-2" />
              Back to Personas
            </Button>
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
          backHref={`/clusters/${clusterName}/personas`}
          backLabel="Back to Personas"
          icon={Users}
          iconColor="text-blue-500"
          title={
            <div className="flex items-center space-x-3">
              <span>{persona.spec.displayName || persona.metadata.name}</span>
              <div className="flex items-center space-x-2">
                {getStatusIcon(persona)}
                <Badge className={getStatusColor(persona)}>
                  {persona.status?.phase || 'Unknown'}
                </Badge>
              </div>
            </div>
          }
          subtitle="LanguagePersona"
          actions={
            <>
              <Button 
                variant="outline"
                onClick={() => router.push(`/clusters/${clusterName}/personas/${personaName}/edit`)}
              >
                <Edit className="h-4 w-4 mr-2" />
                Edit
              </Button>
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button variant="outline" size="icon">
                    <MoreVertical className="h-4 w-4" />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end">
                  <DropdownMenuItem onClick={handleViewYaml}>
                    <FileCode className="h-4 w-4 mr-2" />
                    View YAML
                  </DropdownMenuItem>
                  <DropdownMenuItem 
                    onClick={handleDeletePersona}
                    disabled={deletePersona.isPending}
                    className="text-red-600 focus:text-red-600 focus:bg-red-50"
                  >
                    <Trash2 className="h-4 w-4 mr-2" />
                    {deletePersona.isPending ? 'Deleting...' : 'Delete Persona'}
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </>
          }
        />

        {/* Tabs */}
        <Tabs value={activeTab} onValueChange={setActiveTab} className="space-y-6">
          <TabsList>
            <TabsTrigger value="overview">Overview</TabsTrigger>
            <TabsTrigger value="examples">Examples</TabsTrigger>
          </TabsList>

          <TabsContent value="overview" className="space-y-6">
            <div className="grid gap-6 md:grid-cols-2">
              {/* Basic Info */}
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
                    <p className="text-sm">{persona.spec.displayName}</p>
                  </div>
                  <div>
                    <p className="text-sm font-medium text-muted-foreground">Cluster</p>
                    <p className="text-sm">{clusterName}</p>
                  </div>
                  <div>
                    <p className="text-sm font-medium text-muted-foreground">Namespace</p>
                    <p className="text-sm">{persona.metadata.namespace}</p>
                  </div>
                  {persona.spec.tone && (
                    <div>
                      <p className="text-sm font-medium text-muted-foreground">Tone</p>
                      <Badge variant="secondary">{persona.spec.tone}</Badge>
                    </div>
                  )}
                  {persona.spec.language && (
                    <div>
                      <p className="text-sm font-medium text-muted-foreground">Language</p>
                      <Badge variant="outline">{persona.spec.language}</Badge>
                    </div>
                  )}
                  <div>
                    <p className="text-sm font-medium text-muted-foreground">Created</p>
                    <p className="text-sm">{formatTimeAgo(persona.metadata.creationTimestamp)}</p>
                  </div>
                </CardContent>
              </Card>

              {/* Configuration */}
              <Card>
                <CardHeader>
                  <CardTitle>Configuration</CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div>
                    <p className="text-sm font-medium text-muted-foreground">Description</p>
                    <p className="text-sm">{persona.spec.description}</p>
                  </div>
                  {persona.spec.version && (
                    <div>
                      <p className="text-sm font-medium text-muted-foreground">Version</p>
                      <p className="text-sm">{persona.spec.version}</p>
                    </div>
                  )}
                  <div>
                    <p className="text-sm font-medium text-muted-foreground">Status</p>
                    <Badge variant={persona.status?.phase === 'Ready' ? 'default' : 'secondary'}>
                      {persona.status?.phase || 'Unknown'}
                    </Badge>
                  </div>
                </CardContent>
              </Card>
            </div>

            {/* System Prompt */}
            {persona.spec.systemPrompt && (
              <Card>
                <CardHeader>
                  <CardTitle className="flex items-center space-x-2">
                    <MessageCircle className="h-5 w-5" />
                    <span>System Prompt</span>
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="bg-gray-50 rounded-lg p-4">
                    <pre className="whitespace-pre-wrap text-sm font-mono">
                      {persona.spec.systemPrompt}
                    </pre>
                  </div>
                </CardContent>
              </Card>
            )}

            {/* Capabilities and Limitations */}
            <div className="grid gap-6 md:grid-cols-2">
              <Card>
                <CardHeader>
                  <CardTitle className="flex items-center space-x-2">
                    <Target className="h-5 w-5" />
                    <span>Capabilities ({persona.spec.capabilities?.length || 0})</span>
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  {persona.spec.capabilities && persona.spec.capabilities.length > 0 ? (
                    <div className="space-y-2">
                      {persona.spec.capabilities.map((capability: any, index: number) => (
                        <div key={index} className="flex items-center space-x-2">
                          <div className="w-2 h-2 bg-green-500 rounded-full flex-shrink-0" />
                          <p className="text-sm">{capability}</p>
                        </div>
                      ))}
                    </div>
                  ) : (
                    <p className="text-sm text-muted-foreground">No capabilities specified</p>
                  )}
                </CardContent>
              </Card>

              <Card>
                <CardHeader>
                  <CardTitle className="flex items-center space-x-2">
                    <AlertCircle className="h-5 w-5" />
                    <span>Limitations ({persona.spec.limitations?.length || 0})</span>
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  {persona.spec.limitations && persona.spec.limitations.length > 0 ? (
                    <div className="space-y-2">
                      {persona.spec.limitations.map((limitation: any, index: number) => (
                        <div key={index} className="flex items-center space-x-2">
                          <div className="w-2 h-2 bg-red-500 rounded-full flex-shrink-0" />
                          <p className="text-sm">{limitation}</p>
                        </div>
                      ))}
                    </div>
                  ) : (
                    <p className="text-sm text-muted-foreground">No limitations specified</p>
                  )}
                </CardContent>
              </Card>
            </div>

            {/* Instructions */}
            {persona.spec.instructions && persona.spec.instructions.length > 0 && (
              <Card>
                <CardHeader>
                  <CardTitle className="flex items-center space-x-2">
                    <BookOpen className="h-5 w-5" />
                    <span>Instructions ({persona.spec.instructions.length})</span>
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="space-y-3">
                    {persona.spec.instructions.map((instruction: any, index: number) => (
                      <div key={index} className="p-3 border rounded-lg">
                        <p className="text-sm">{instruction}</p>
                      </div>
                    ))}
                  </div>
                </CardContent>
              </Card>
            )}
          </TabsContent>

          <TabsContent value="examples" className="space-y-6">
            {persona.spec.examples && persona.spec.examples.length > 0 ? (
              <div className="space-y-4">
                {persona.spec.examples.map((example: any, index: number) => (
                  <Card key={index}>
                    <CardHeader>
                      <CardTitle className="text-lg">Example {index + 1}</CardTitle>
                      {example.tags && example.tags.length > 0 && (
                        <div className="flex flex-wrap gap-1 mt-2">
                          {example.tags.map((tag: any, tagIndex: number) => (
                            <Badge key={tagIndex} variant="outline" className="text-xs">
                              {tag}
                            </Badge>
                          ))}
                        </div>
                      )}
                    </CardHeader>
                    <CardContent className="space-y-4">
                      {example.context && (
                        <div>
                          <p className="text-sm font-medium text-muted-foreground mb-2">Context</p>
                          <p className="text-sm bg-blue-50 p-3 rounded-lg">{example.context}</p>
                        </div>
                      )}
                      <div>
                        <p className="text-sm font-medium text-muted-foreground mb-2">Input</p>
                        <div className="bg-gray-50 rounded-lg p-3">
                          <pre className="whitespace-pre-wrap text-sm font-mono">{example.input}</pre>
                        </div>
                      </div>
                      <div>
                        <p className="text-sm font-medium text-muted-foreground mb-2">Expected Output</p>
                        <div className="bg-green-50 rounded-lg p-3">
                          <pre className="whitespace-pre-wrap text-sm font-mono">{example.output}</pre>
                        </div>
                      </div>
                    </CardContent>
                  </Card>
                ))}
              </div>
            ) : (
              <Card>
                <CardContent className="flex flex-col items-center justify-center py-16">
                  <BookOpen className="h-16 w-16 text-gray-400 mb-4" />
                  <CardTitle className="text-xl mb-2">No examples</CardTitle>
                  <CardDescription className="text-center max-w-md">
                    No conversation examples have been configured for this persona.
                  </CardDescription>
                </CardContent>
              </Card>
            )}
          </TabsContent>
        </Tabs>

        {/* Persona Events */}
        <ResourceEventsActivity
          resourceType="persona"
          resourceName={personaName}
          namespace={persona.metadata.namespace}
          limit={15}
        />
      </div>

      {/* YAML Modal */}
      <Dialog open={yamlModalOpen} onOpenChange={setYamlModalOpen}>
        <DialogContent className="w-[85vw] !max-w-[85vw] max-h-[85vh] flex flex-col sm:!max-w-[85vw] md:!max-w-[85vw] lg:!max-w-[85vw]">
          <DialogHeader>
            <DialogTitle>LanguagePersona YAML</DialogTitle>
          </DialogHeader>
          
          <div className="flex-1 min-h-0 flex flex-col">
            {yamlLoading ? (
              <div className="flex items-center justify-center py-8">
                <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-gray-900"></div>
              </div>
            ) : (
              <div className="border rounded-lg flex-1 min-h-0 flex flex-col">
                <div className="bg-muted p-3 border-b flex justify-between items-center">
                  <span className="font-medium text-sm">
                    {persona?.metadata.name || personaName}.yaml
                  </span>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={handleCopyYaml}
                    disabled={!yamlContent || yamlContent.startsWith('Error')}
                  >
                    {copied ? (
                      <>
                        <Check className="h-4 w-4 mr-2" />
                        Copied!
                      </>
                    ) : (
                      <>
                        <Copy className="h-4 w-4 mr-2" />
                        Copy
                      </>
                    )}
                  </Button>
                </div>
                <div className="flex-1 min-h-0 overflow-auto">
                  {yamlContent.startsWith('Error') ? (
                    <div className="p-4 text-red-600 font-mono text-sm">
                      {yamlContent}
                    </div>
                  ) : (
                    <SyntaxHighlighter
                      language="yaml"
                      style={oneLight}
                      customStyle={{
                        margin: 0,
                        padding: '1rem',
                        background: 'transparent',
                        fontSize: '0.875rem',
                        lineHeight: '1.5',
                        height: '100%',
                        overflow: 'auto'
                      }}
                      codeTagProps={{
                        style: {
                          fontFamily: 'ui-monospace, SFMono-Regular, "SF Mono", Monaco, Consolas, "Liberation Mono", "Courier New", monospace'
                        }
                      }}
                    >
                      {yamlContent}
                    </SyntaxHighlighter>
                  )}
                </div>
              </div>
            )}
          </div>
        </DialogContent>
      </Dialog>
    </AuthenticatedLayout>
  )
}