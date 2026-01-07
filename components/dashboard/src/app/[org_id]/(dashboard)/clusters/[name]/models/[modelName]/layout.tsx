'use client'

import { useState } from 'react'
import { useRouter, useParams, usePathname } from 'next/navigation'
import Link from 'next/link'
import { Cpu, Edit, MoreVertical, FileCode, Trash2, Home, Info, Globe, ScrollText, Copy, Check } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from '@/components/ui/dropdown-menu'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Skeleton } from '@/components/ui/skeleton'
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter'
import { oneLight, oneDark } from 'react-syntax-highlighter/dist/esm/styles/prism'
import { useTheme } from 'next-themes'
import { useModel, useDeleteModel } from '@/hooks/use-models'
import { fetchWithOrganization } from '@/lib/api-client'
import { useOrganization } from '@/components/organization-provider'
import { ResourceHeader } from '@/components/ui/resource-header'
import { NotFound } from '@/components/ui/not-found'
import { cn } from '@/lib/utils'

function getCurrentTabValue(pathname: string, clusterName: string, modelName: string): string {
  // Check for organization-scoped paths and non-organization paths
  const orgBasePath = `/clusters/${clusterName}/models/${modelName}`
  const paths = [pathname, pathname.replace(/^\/[^/]+/, '')]
  
  for (const path of paths) {
    if (path === orgBasePath || path === `/clusters/${clusterName}/models/${modelName}`) return 'overview'
    if (path.endsWith('/details')) return 'details'
    if (path.endsWith('/network')) return 'network'
    if (path.endsWith('/logs')) return 'logs'
  }
  
  return 'overview'
}

function getCurrentTabValueOld(pathname: string, clusterName: string, modelName: string): string {
  const basePath = `/clusters/${clusterName}/models/${modelName}`
  
  if (pathname === basePath) return 'overview'
  if (pathname.endsWith('/details')) return 'details'
  if (pathname.endsWith('/network')) return 'network'
  if (pathname.endsWith('/logs')) return 'logs'
  
  return 'overview'
}

interface ModelDetailLayoutProps {
  children: React.ReactNode
}

export default function ModelDetailLayout({ children }: ModelDetailLayoutProps) {
  const router = useRouter()
  const params = useParams()
  const pathname = usePathname()
  const { getOrgUrl } = useOrganization()
  const clusterName = params?.name as string
  const modelName = params?.modelName as string

  const [yamlModalOpen, setYamlModalOpen] = useState(false)
  const [yamlContent, setYamlContent] = useState('')
  const [yamlLoading, setYamlLoading] = useState(false)
  const [copied, setCopied] = useState(false)
  const { theme } = useTheme()

  const { data: modelResponse, isLoading, error } = useModel(modelName, clusterName)
  const deleteModel = useDeleteModel(clusterName)

  const model = modelResponse?.data
  
  // Check if we're on the edit page
  const isEditPage = pathname.endsWith('/edit')

  const handleDeleteModel = async () => {
    if (!model || !model.metadata.name) return

    if (confirm(`Are you sure you want to delete model "${model.metadata.name}"?`)) {
      try {
        await deleteModel.mutateAsync(model.metadata.name)
        router.push(getOrgUrl(`/clusters/${clusterName}/models`))
      } catch (error) {
        console.error('Failed to delete model:', error)
        alert('Failed to delete model. Please try again.')
      }
    }
  }

  const handleViewYaml = async () => {
    setYamlModalOpen(true)
    setYamlLoading(true)
    try {
      const response = await fetchWithOrganization(`/api/clusters/${clusterName}/models/${modelName}/yaml`)
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
      <div className="space-y-6">
        <div className="flex items-center space-x-4">
          <Skeleton className="h-10 w-16" />
          <Skeleton className="h-8 w-8" />
          <div className="space-y-2">
            <Skeleton className="h-6 w-48" />
            <Skeleton className="h-4 w-32" />
          </div>
        </div>
        <Skeleton className="h-64 w-full" />
      </div>
    )
  }

  if (error || !model) {
    // Determine if it's a 404 error or other error type
    const is404Error = error && (error as any)?.status === 404
    const errorMessage = error ? (error as Error).message : `Model "${modelName}" could not be found in cluster "${clusterName}".`

    return (
      <NotFound
          title={is404Error ? 'Model Not Found' : 'Error Loading Model'}
          message={errorMessage}
          onBack={() => router.push(getOrgUrl(`/clusters/${clusterName}/models`))}
          backLabel="Back to Models"
        />
    )
  }

  // Render edit page layout differently
  if (isEditPage) {
    return (
      <div className="space-y-6">
        {/* Edit Page Header */}
        <ResourceHeader
          backHref={getOrgUrl(`/clusters/${clusterName}/models/${modelName}`)}
          backLabel="Back to Model"
          icon={Cpu}
          title={`Edit ${model.metadata.name}`}
          subtitle="LanguageModel"
        />
        
        {/* Page Content */}
        {children}
      </div>
    )
  }

  // Regular model detail layout
  return (
      <div className="space-y-6">
        {/* Header */}
        <ResourceHeader
          backHref={getOrgUrl(`/clusters/${clusterName}/models`)}
          backLabel="Back to Models"
          icon={Cpu}
          title={
            <div className="flex items-center space-x-3">
              <span>{model.metadata.name}</span>
              <div className="flex items-center space-x-2">
                <span className={`text-[10px] tracking-wider uppercase font-light`}>
                  {model.status?.phase || 'Unknown'}
                </span>
              </div>
            </div>
          }
          subtitle="LanguageModel"
          actions={
            <>
              <Button
                variant="outline"
                onClick={() => router.push(getOrgUrl(`/clusters/${clusterName}/models/${modelName}/edit`))}
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
                    onClick={handleDeleteModel}
                    disabled={deleteModel.isPending}
                    className="text-red-600 focus:text-red-600 focus:bg-red-50"
                  >
                    <Trash2 className="h-4 w-4 mr-2" />
                    {deleteModel.isPending ? 'Deleting...' : 'Delete Model'}
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </>
          }
        />

        {/* Tabs Navigation */}
        <Tabs value={getCurrentTabValue(pathname, clusterName, modelName)}>
          <TabsList className="w-full">
            <TabsTrigger value="overview" asChild>
              <Link href={getOrgUrl(`/clusters/${clusterName}/models/${modelName}`)}>
                <Home className="w-4 h-4 mr-2" />
                Overview
              </Link>
            </TabsTrigger>
            <TabsTrigger value="details" asChild>
              <Link href={getOrgUrl(`/clusters/${clusterName}/models/${modelName}/details`)}>
                <Info className="w-4 h-4 mr-2" />
                Details
              </Link>
            </TabsTrigger>
            <TabsTrigger value="network" asChild>
              <Link href={getOrgUrl(`/clusters/${clusterName}/models/${modelName}/network`)}>
                <Globe className="w-4 h-4 mr-2" />
                Network
              </Link>
            </TabsTrigger>
            <TabsTrigger value="logs" asChild>
              <Link href={getOrgUrl(`/clusters/${clusterName}/models/${modelName}/logs`)}>
                <ScrollText className="w-4 h-4 mr-2" />
                Logs
              </Link>
            </TabsTrigger>
          </TabsList>
        </Tabs>

        {/* Page Content */}
        {children}

        {/* YAML Modal */}
      <Dialog open={yamlModalOpen} onOpenChange={setYamlModalOpen}>
        <DialogContent className="w-[85vw] !max-w-[85vw] max-h-[85vh] flex flex-col sm:!max-w-[85vw] md:!max-w-[85vw] lg:!max-w-[85vw]">
          <DialogHeader>
            <DialogTitle>LanguageModel YAML</DialogTitle>
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
                    {model?.metadata.name || modelName}.yaml
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
                      style={theme === 'dark' ? oneDark : oneLight}
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
      </div>
  )
}