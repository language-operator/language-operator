'use client'

import { useState } from 'react'
import { useRouter, useParams } from 'next/navigation'
import { AuthenticatedLayout } from '@/components/layout/authenticated-layout'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { ArrowLeft, Edit, Trash2, Play, Pause, Wrench, MoreVertical, FileCode, Copy, Check } from 'lucide-react'
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter'
import { oneLight } from 'react-syntax-highlighter/dist/esm/styles/prism'
import { useQuery } from '@tanstack/react-query'
import { ResourceHeader } from '@/components/ui/resource-header'
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from '@/components/ui/dropdown-menu'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'

export default function ClusterToolDetailPage() {
  const router = useRouter()
  const params = useParams()
  const clusterName = params?.name as string
  const toolName = params?.toolName as string
  const [yamlModalOpen, setYamlModalOpen] = useState(false)
  const [yamlContent, setYamlContent] = useState('')
  const [yamlLoading, setYamlLoading] = useState(false)
  const [copied, setCopied] = useState(false)

  // Use cluster-scoped API endpoint for tool details
  const { data: toolResponse, isLoading } = useQuery({
    queryKey: ['cluster-tool', clusterName, toolName],
    queryFn: async () => {
      const response = await fetch(`/api/clusters/${clusterName}/tools/${toolName}`)
      if (!response.ok) {
        throw new Error('Failed to fetch tool')
      }
      return response.json()
    },
    enabled: !!clusterName && !!toolName,
    refetchInterval: 5000,
  })
  const tool = toolResponse?.data

  const handleEdit = () => {
    router.push(`/clusters/${clusterName}/tools/${toolName}/edit`)
  }

  const handleDelete = async () => {
    if (!confirm(`Are you sure you want to delete the tool "${toolName}"?`)) {
      return
    }

    try {
      const response = await fetch(`/api/clusters/${clusterName}/tools/${toolName}`, {
        method: 'DELETE',
      })

      if (!response.ok) {
        throw new Error('Failed to delete tool')
      }

      router.push(`/clusters/${clusterName}/tools`)
    } catch (error) {
      console.error('Error deleting tool:', error)
      alert('Failed to delete tool')
    }
  }

  const handleBack = () => {
    router.push(`/clusters/${clusterName}/tools`)
  }

  const handleViewYaml = async () => {
    setYamlModalOpen(true)
    setYamlLoading(true)
    try {
      const response = await fetch(`/api/clusters/${clusterName}/tools/${toolName}/yaml`)
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
            <div>
              <div className="h-8 w-48 bg-gray-200 rounded animate-pulse"></div>
              <div className="h-4 w-32 bg-gray-200 rounded mt-2 animate-pulse"></div>
            </div>
          </div>
        </div>
      </AuthenticatedLayout>
    )
  }

  if (!tool) {
    return (
      <AuthenticatedLayout>
        <div className="space-y-6">
          <div className="flex items-center space-x-4">
            <Button variant="outline" size="icon" onClick={handleBack}>
              <ArrowLeft className="h-4 w-4" />
            </Button>
            <div>
              <h1 className="text-3xl font-bold">Tool Not Found</h1>
              <p className="text-muted-foreground mt-1">
                The tool "{toolName}" was not found in cluster "{clusterName}"
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
        <ResourceHeader
          backHref={`/clusters/${clusterName}/tools`}
          backLabel="Back to Tools"
          icon={Wrench}
          title={tool.metadata.name}
          subtitle="LanguageTool"
          actions={
            <>
              <Button variant="outline" onClick={handleEdit}>
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
                    onClick={handleDelete}
                    className="text-red-600 focus:text-red-600 focus:bg-red-50"
                  >
                    <Trash2 className="h-4 w-4 mr-2" />
                    Delete Tool
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </>
          }
        />

        {/* Tool Details */}
        <div className="grid gap-6">
          {/* Overview */}
          <Card>
            <CardHeader>
              <CardTitle>Overview</CardTitle>
              <CardDescription>Basic tool information and status</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <p className="text-sm font-medium">Name</p>
                  <p className="text-sm text-muted-foreground">{tool.metadata.name}</p>
                </div>
                <div>
                  <p className="text-sm font-medium">Type</p>
                  <p className="text-sm text-muted-foreground">{tool.spec.type || 'Unknown'}</p>
                </div>
                <div>
                  <p className="text-sm font-medium">Status</p>
                  <Badge variant={tool.status?.phase === 'Ready' ? 'default' : 'secondary'}>
                    {tool.status?.phase || 'Unknown'}
                  </Badge>
                </div>
                <div>
                  <p className="text-sm font-medium">Created</p>
                  <p className="text-sm text-muted-foreground">
                    {tool.metadata.creationTimestamp ? new Date(tool.metadata.creationTimestamp).toLocaleDateString() : 'Unknown'}
                  </p>
                </div>
              </div>

              {tool.spec.description && (
                <div>
                  <p className="text-sm font-medium">Description</p>
                  <p className="text-sm text-muted-foreground">{tool.spec.description}</p>
                </div>
              )}
            </CardContent>
          </Card>

          {/* Configuration */}
          <Card>
            <CardHeader>
              <CardTitle>Configuration</CardTitle>
              <CardDescription>Tool configuration settings</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                {tool.spec.image && (
                  <div>
                    <p className="text-sm font-medium">Image</p>
                    <p className="text-sm text-muted-foreground font-mono">{tool.spec.image}</p>
                  </div>
                )}

                {tool.spec.endpoint && (
                  <div>
                    <p className="text-sm font-medium">Endpoint</p>
                    <p className="text-sm text-muted-foreground font-mono">{tool.spec.endpoint}</p>
                  </div>
                )}

                {tool.spec.port && (
                  <div>
                    <p className="text-sm font-medium">Port</p>
                    <p className="text-sm text-muted-foreground">{tool.spec.port}</p>
                  </div>
                )}

                {tool.spec.resources && (
                  <div>
                    <p className="text-sm font-medium">Resource Limits</p>
                    <div className="grid grid-cols-2 gap-2 text-sm text-muted-foreground">
                      {tool.spec.resources.requests && (
                        <div>
                          <span className="font-mono">CPU Request: {tool.spec.resources.requests.cpu}</span><br />
                          <span className="font-mono">Memory Request: {tool.spec.resources.requests.memory}</span>
                        </div>
                      )}
                      {tool.spec.resources.limits && (
                        <div>
                          <span className="font-mono">CPU Limit: {tool.spec.resources.limits.cpu}</span><br />
                          <span className="font-mono">Memory Limit: {tool.spec.resources.limits.memory}</span>
                        </div>
                      )}
                    </div>
                  </div>
                )}
              </div>
            </CardContent>
          </Card>
        </div>
      </div>

      {/* YAML Modal */}
      <Dialog open={yamlModalOpen} onOpenChange={setYamlModalOpen}>
        <DialogContent className="w-[85vw] !max-w-[85vw] max-h-[85vh] flex flex-col sm:!max-w-[85vw] md:!max-w-[85vw] lg:!max-w-[85vw]">
          <DialogHeader>
            <DialogTitle>LanguageTool YAML</DialogTitle>
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
                    {tool?.metadata.name || toolName}.yaml
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