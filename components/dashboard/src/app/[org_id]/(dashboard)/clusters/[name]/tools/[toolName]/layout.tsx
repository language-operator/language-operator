'use client'

import { useState } from 'react'
import { useRouter, useParams, usePathname } from 'next/navigation'
import Link from 'next/link'
import { Wrench, Edit, MoreVertical, FileCode, Trash2, Home, ScrollText, Copy, Check } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from '@/components/ui/dropdown-menu'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Skeleton } from '@/components/ui/skeleton'
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter'
import { oneLight, oneDark } from 'react-syntax-highlighter/dist/esm/styles/prism'
import { useTheme } from 'next-themes'
import { useTool, useDeleteTool } from '@/hooks/use-tools'
import { fetchWithOrganization } from '@/lib/api-client'
import { useOrganization } from '@/components/organization-provider'
import { ResourceHeader } from '@/components/ui/resource-header'
import { NotFound } from '@/components/ui/not-found'
import { cn } from '@/lib/utils'

function getCurrentTabValue(pathname: string, clusterName: string, toolName: string): string {
  const basePath = `/clusters/${clusterName}/tools/${toolName}`
  
  if (pathname === basePath) return 'overview'
  if (pathname.endsWith('/logs')) return 'logs'
  
  return 'overview'
}

interface ToolDetailLayoutProps {
  children: React.ReactNode
}

export default function ToolDetailLayout({ children }: ToolDetailLayoutProps) {
  const router = useRouter()
  const params = useParams()
  const pathname = usePathname()
  const { getOrgUrl } = useOrganization()
  const clusterName = params?.name as string
  const toolName = params?.toolName as string

  const [yamlModalOpen, setYamlModalOpen] = useState(false)
  const [yamlContent, setYamlContent] = useState('')
  const [yamlLoading, setYamlLoading] = useState(false)
  const [copied, setCopied] = useState(false)
  const { theme } = useTheme()

  const { data: toolResponse, isLoading, error } = useTool(toolName, clusterName)
  const deleteTool = useDeleteTool(clusterName)

  const tool = toolResponse?.data
  
  // Check if we're on the edit page
  const isEditPage = pathname.endsWith('/edit')

  const handleDeleteTool = async () => {
    if (!tool || !tool.metadata.name) return

    if (confirm(`Are you sure you want to delete tool "${tool.metadata.name}"?`)) {
      try {
        await deleteTool.mutateAsync(tool.metadata.name)
        router.push(getOrgUrl(`/clusters/${clusterName}/tools`))
      } catch (error) {
        console.error('Failed to delete tool:', error)
        alert('Failed to delete tool. Please try again.')
      }
    }
  }

  const handleViewYaml = async () => {
    setYamlModalOpen(true)
    setYamlLoading(true)
    try {
      const response = await fetchWithOrganization(`/api/clusters/${clusterName}/tools/${toolName}/yaml`)
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

  if (error || !tool) {
    // Determine if it's a 404 error or other error type
    const is404Error = error && (error as any)?.status === 404
    const errorMessage = error ? (error as Error).message : `Tool "${toolName}" could not be found in cluster "${clusterName}".`

    return (
        <NotFound
          title={is404Error ? 'Tool Not Found' : 'Error Loading Tool'}
          message={errorMessage}
          onBack={() => router.push(getOrgUrl(`/clusters/${clusterName}/tools`))}
          backLabel="Back to Tools"
        />
    )
  }

  // Render edit page layout differently
  if (isEditPage) {
    return (
        <div className="space-y-6">
          {/* Edit Page Header */}
          <ResourceHeader
            backHref={getOrgUrl(`/clusters/${clusterName}/tools/${toolName}`)}
            backLabel="Back to Tool"
            icon={Wrench}
            title={`Edit ${tool.metadata.name}`}
            subtitle="LanguageTool"
          />
          
          {/* Page Content */}
          {children}
        </div>
    )
  }

  // Regular tool detail layout
  return (
      <div className="space-y-6">
        {/* Header */}
        <ResourceHeader
          backHref={getOrgUrl(`/clusters/${clusterName}/tools`)}
          backLabel="Back to Tools"
          icon={Wrench}
          title={
            <div className="flex items-center space-x-3">
              <span>{tool.metadata.name}</span>
              <div className="flex items-center space-x-2">
                <span className={`text-[10px] tracking-wider uppercase font-light`}>
                  {tool.status?.phase || 'Unknown'}
                </span>
              </div>
            </div>
          }
          subtitle="LanguageTool"
          actions={
            <>
              <Button
                variant="outline"
                onClick={() => router.push(getOrgUrl(`/clusters/${clusterName}/tools/${toolName}/edit`))}
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
                    onClick={handleDeleteTool}
                    disabled={deleteTool.isPending}
                    className="text-red-600 focus:text-red-600 focus:bg-red-50"
                  >
                    <Trash2 className="h-4 w-4 mr-2" />
                    {deleteTool.isPending ? 'Deleting...' : 'Delete Tool'}
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </>
          }
        />

        {/* Tabs Navigation */}
        <Tabs value={getCurrentTabValue(pathname, clusterName, toolName)}>
          <TabsList className="w-full">
            <TabsTrigger value="overview" asChild>
              <Link href={getOrgUrl(`/clusters/${clusterName}/tools/${toolName}`)}>
                <Home className="w-4 h-4 mr-2" />
                Overview
              </Link>
            </TabsTrigger>
            <TabsTrigger value="logs" asChild>
              <Link href={getOrgUrl(`/clusters/${clusterName}/tools/${toolName}/logs`)}>
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