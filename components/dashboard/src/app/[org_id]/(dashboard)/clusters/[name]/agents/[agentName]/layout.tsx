'use client'

import { useState } from 'react'
import { useRouter, useParams, usePathname } from 'next/navigation'
import Link from 'next/link'
import { Bot, Edit, MoreVertical, FileCode, Trash2, Play, Home, Code, FolderOpen, ScrollText, Clock, Copy, Check, MessageCircle, Activity, BarChart3, DollarSign } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from '@/components/ui/dropdown-menu'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog'
import { Skeleton } from '@/components/ui/skeleton'
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter'
import { oneLight, oneDark } from 'react-syntax-highlighter/dist/esm/styles/prism'
import { useTheme } from 'next-themes'
import { useAgent, useDeleteAgent } from '@/hooks/use-agents'
import { fetchWithOrganization } from '@/lib/api-client'
import { useOrganization } from '@/components/organization-provider'
import { ResourceHeader } from '@/components/ui/resource-header'
import { NotFound } from '@/components/ui/not-found'
import { cn } from '@/lib/utils'
import { getStatusIcon, getStatusColor } from '@/components/agents/utils'

function getCurrentTabValue(pathname: string, clusterName: string, agentName: string): string {
  // Check for organization-scoped paths and non-organization paths
  const orgBasePath = `/clusters/${clusterName}/agents/${agentName}`
  const paths = [pathname, pathname.replace(/^\/[^/]+/, '')]
  
  for (const path of paths) {
    if (path === orgBasePath || path === `/clusters/${clusterName}/agents/${agentName}`) return 'overview'
    if (path.endsWith('/code')) return 'code'
    if (path.endsWith('/workspace')) return 'workspace'
    if (path.endsWith('/logs')) return 'logs'
    if (path.endsWith('/traces')) return 'traces'
    if (path.endsWith('/cost')) return 'cost'
  }
  
  return 'overview'
}

interface AgentDetailLayoutProps {
  children: React.ReactNode
}

export default function AgentDetailLayout({ children }: AgentDetailLayoutProps) {
  const router = useRouter()
  const params = useParams()
  const pathname = usePathname()
  const { getOrgUrl } = useOrganization()
  const clusterName = params?.name as string
  const agentName = params?.agentName as string

  const [yamlModalOpen, setYamlModalOpen] = useState(false)
  const [yamlContent, setYamlContent] = useState('')
  const [yamlLoading, setYamlLoading] = useState(false)
  const [copied, setCopied] = useState(false)
  const [isExecuting, setIsExecuting] = useState(false)
  const [showExecuteDialog, setShowExecuteDialog] = useState(false)
  const { theme } = useTheme()

  const { data: agentResponse, isLoading, error } = useAgent(agentName, clusterName)
  const deleteAgent = useDeleteAgent(clusterName)

  const agent = agentResponse?.data
  
  // Check if we're on the edit page
  const isEditPage = pathname.endsWith('/edit')


  const handleDeleteAgent = async () => {
    if (!agent || !agent.metadata.name) return

    if (confirm(`Are you sure you want to delete agent "${agent.metadata.name}"?`)) {
      try {
        await deleteAgent.mutateAsync(agent.metadata.name)
        router.push(getOrgUrl(`/clusters/${clusterName}/agents`))
      } catch (error) {
        console.error('Failed to delete agent:', error)
        alert('Failed to delete agent. Please try again.')
      }
    }
  }


  const handleViewYaml = async () => {
    setYamlModalOpen(true)
    setYamlLoading(true)
    try {
      const response = await fetchWithOrganization(`/api/clusters/${clusterName}/agents/${agentName}/yaml`)
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

  const handleRunManually = () => {
    setShowExecuteDialog(true)
  }

  const handleExecuteConfirm = async () => {
    if (!agent || !agent.metadata.name) return

    try {
      setIsExecuting(true)
      setShowExecuteDialog(false)

      const response = await fetchWithOrganization(`/api/clusters/${clusterName}/agents/${agentName}/execute`, {
        method: 'POST',
      })

      if (!response.ok) {
        const errorData = await response.json()
        throw new Error(errorData.message || 'Failed to execute agent')
      }

      const result = await response.json()
      console.log('Manual execution started:', result)

      // Wait a moment for the job to start
      setTimeout(() => {
        // Navigate to logs page
        router.push(getOrgUrl(`/clusters/${clusterName}/agents/${agentName}/logs`))
        setIsExecuting(false)
      }, 2000)

    } catch (error) {
      console.error('Failed to execute agent manually:', error)
      alert('Failed to execute agent. Please try again.')
      setIsExecuting(false)
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

  if (error || !agent) {
    // Determine if it's a 404 error or other error type
    const is404Error = error && (error as any)?.status === 404
    const errorMessage = error ? (error as Error).message : `Agent "${agentName}" could not be found in cluster "${clusterName}".`

    return (
      <NotFound
        title={is404Error ? 'Agent Not Found' : 'Error Loading Agent'}
        message={errorMessage}
        onBack={() => router.push(getOrgUrl(`/clusters/${clusterName}/agents`))}
        backLabel="Back to Agents"
      />
    )
  }

  // Render edit page layout differently
  if (isEditPage) {
    return (
      <div className="space-y-6">
        {/* Edit Page Header */}
        <ResourceHeader
          backHref={getOrgUrl(`/clusters/${clusterName}/agents/${agentName}`)}
          backLabel="Back to Agent"
          icon={Bot}
          title={`Edit ${agent.metadata.name}`}
          subtitle="LanguageAgent"
        />
        
        {/* Page Content */}
        {children}
      </div>
    )
  }

  // Regular agent detail layout
  return (
      <div className="space-y-6">
        {/* Header */}
        <ResourceHeader
          backHref={getOrgUrl(`/clusters/${clusterName}/agents`)}
          backLabel="Back to Agents"
          icon={Bot}
          title={
            <div className="flex items-center space-x-3">
              <span>{agent.metadata.name}</span>
              <div className="flex items-center space-x-2">
                {getStatusIcon(agent)}
                <span className={`text-[10px] tracking-wider uppercase font-light ${getStatusColor(agent)}`}>
                  {agent.status?.phase || 'Unknown'}
                </span>
              </div>
            </div>
          }
          subtitle="LanguageAgent"
          actions={
            <>
              <Button
                variant="outline"
                onClick={() => router.push(getOrgUrl(`/clusters/${clusterName}/agents/${agentName}/edit`))}
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
                  <DropdownMenuItem asChild>
                    <Link href={getOrgUrl(`/clusters/${clusterName}/console?agent=${agentName}`)}>
                      <MessageCircle className="h-4 w-4 mr-2" />
                      Open in Console
                    </Link>
                  </DropdownMenuItem>
                  {agent?.spec?.executionMode === 'scheduled' && (
                    <DropdownMenuItem
                      onClick={handleRunManually}
                      disabled={isExecuting}
                    >
                      <Play className="h-4 w-4 mr-2" />
                      {isExecuting ? 'Executing...' : 'Run Manually'}
                    </DropdownMenuItem>
                  )}
                  <DropdownMenuItem onClick={handleViewYaml}>
                    <FileCode className="h-4 w-4 mr-2" />
                    View YAML
                  </DropdownMenuItem>
                  <DropdownMenuItem
                    onClick={handleDeleteAgent}
                    disabled={deleteAgent.isPending}
                    className="text-red-600 focus:text-red-600 focus:bg-red-50"
                  >
                    <Trash2 className="h-4 w-4 mr-2" />
                    {deleteAgent.isPending ? 'Deleting...' : 'Delete Agent'}
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </>
          }
        />

        {/* Tabs Navigation */}
        <Tabs value={getCurrentTabValue(pathname, clusterName, agentName)}>
          <TabsList className="w-full">
            <TabsTrigger value="overview" asChild>
              <Link href={getOrgUrl(`/clusters/${clusterName}/agents/${agentName}`)}>
                <Home className="w-4 h-4 mr-2" />
                Overview
              </Link>
            </TabsTrigger>
            <TabsTrigger value="code" asChild>
              <Link href={getOrgUrl(`/clusters/${clusterName}/agents/${agentName}/code`)}>
                <Code className="w-4 h-4 mr-2" />
                Code
              </Link>
            </TabsTrigger>
            <TabsTrigger value="cost" asChild>
              <Link href={getOrgUrl(`/clusters/${clusterName}/agents/${agentName}/cost`)}>
                <DollarSign className="w-4 h-4 mr-2" />
                Cost
              </Link>
            </TabsTrigger>
            <TabsTrigger value="workspace" asChild>
              <Link href={getOrgUrl(`/clusters/${clusterName}/agents/${agentName}/workspace`)}>
                <FolderOpen className="w-4 h-4 mr-2" />
                Workspace
              </Link>
            </TabsTrigger>
            <TabsTrigger value="logs" asChild>
              <Link href={getOrgUrl(`/clusters/${clusterName}/agents/${agentName}/logs`)}>
                <ScrollText className="w-4 h-4 mr-2" />
                Logs
              </Link>
            </TabsTrigger>
            <TabsTrigger value="traces" asChild>
              <Link href={getOrgUrl(`/clusters/${clusterName}/agents/${agentName}/traces`)}>
                <Activity className="w-4 h-4 mr-2" />
                Traces
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
            <DialogTitle>LanguageAgent YAML</DialogTitle>
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
                    {agent?.metadata.name || agentName}.yaml
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

      {/* Manual Execution Confirmation Dialog */}
      <Dialog open={showExecuteDialog} onOpenChange={setShowExecuteDialog}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Run Agent Manually</DialogTitle>
          </DialogHeader>
          <div className="py-4">
            <p className="text-sm text-stone-600 dark:text-stone-400">
              This will create a one-time job to execute the agent "{agentName}" immediately.
              The execution will be independent of the scheduled runs.
            </p>
          </div>
          <DialogFooter className="flex justify-end space-x-2">
            <Button
              variant="outline"
              onClick={() => setShowExecuteDialog(false)}
            >
              Cancel
            </Button>
            <Button
              onClick={handleExecuteConfirm}
              disabled={isExecuting}
            >
              {isExecuting ? 'Starting...' : 'Run Now'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
      </div>
  )
}
