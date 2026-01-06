'use client'

import { useParams, useRouter } from 'next/navigation'
import { useOrganization } from '@/components/organization-provider'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { ResourceHeader } from '@/components/ui/resource-header'
import { Wrench, Download, CheckCircle, Search, ExternalLink, Shield, Network } from 'lucide-react'
import Link from 'next/link'
import { useEffect, useState, useMemo } from 'react'
import { ToolCatalog, ToolCatalogEntry, InstalledTool } from '@/types/tool-catalog'
import { EventsActivity } from '@/components/ui/events-activity'
import { useTools } from '@/hooks/use-tools'
import { useWatchTools } from '@/hooks/use-watch'

export default function ClusterTools() {
  const params = useParams()
  const { getOrgUrl } = useOrganization()
  const clusterName = params?.name as string
  const [catalog, setCatalog] = useState<ToolCatalog | null>(null)
  const [catalogLoading, setCatalogLoading] = useState(true)
  const [catalogError, setCatalogError] = useState<string | null>(null)
  const [searchQuery, setSearchQuery] = useState('')

  // Use the tools hook for installed tools (with real-time updates)
  const { data: toolsData, isLoading: toolsLoading, error: toolsError } = useTools({
    clusterName,
    page: 1,
    limit: 1000, // Get all tools for catalog view
  })

  // Enable real-time updates via SSE watch
  useWatchTools()

  // Convert LanguageTool objects to InstalledTool format
  const installedTools = useMemo(() => {
    return (toolsData?.data || []).map((tool: any) => ({
      name: tool.metadata.name,
      catalogName: tool.metadata.labels?.['langop.io/catalog-name'] || tool.metadata.name,
      status: {
        phase: tool.status?.phase || 'Unknown',
        message: tool.status?.conditions?.[0]?.message || ''
      }
    }))
  }, [toolsData])

  // Fetch catalog separately (static data, doesn't need watch)
  useEffect(() => {
    const fetchCatalog = async () => {
      try {
        setCatalogLoading(true)
        setCatalogError(null)

        const catalogResponse = await fetch('/api/tools/catalog')
        if (!catalogResponse.ok) {
          throw new Error('Failed to fetch tool catalog')
        }
        const catalogData = await catalogResponse.json()
        setCatalog(catalogData)
      } catch (err) {
        console.error('Error fetching catalog:', err)
        setCatalogError(err instanceof Error ? err.message : 'Failed to load tool catalog')
      } finally {
        setCatalogLoading(false)
      }
    }

    fetchCatalog()
  }, [])

  const isToolInstalled = (toolName: string) => {
    return installedTools.some((tool: InstalledTool) =>
      tool.catalogName === toolName ||
      tool.name === toolName
    )
  }

  const getCatalogEntryForInstalledTool = (installedTool: InstalledTool) => {
    if (!catalog?.tools) return null
    const toolName = installedTool.catalogName || installedTool.name
    return Object.entries(catalog.tools).find(([id, _]) => id === toolName)?.[1] || null
  }

  const ToolCard = ({ 
    toolId, 
    tool, 
    isInstalled, 
    installedTool, 
    clusterName 
  }: {
    toolId: string
    tool: ToolCatalogEntry
    isInstalled: boolean
    installedTool?: InstalledTool
    clusterName: string
  }) => {
    const router = useRouter()

    const handleCardClick = (event: React.MouseEvent) => {
      // Only make installed tools clickable
      if (!isInstalled || !installedTool) return
      
      // Don't navigate if clicking on buttons or other interactive elements
      const target = event.target as HTMLElement
      if (target.closest('button') || target.closest('a')) {
        return
      }
      
      router.push(getOrgUrl(`/clusters/${clusterName}/tools/${installedTool.name}`))
    }

    const handleKeyDown = (event: React.KeyboardEvent) => {
      if (!isInstalled || !installedTool) return
      
      if (event.key === 'Enter' || event.key === ' ') {
        event.preventDefault()
        router.push(getOrgUrl(`/clusters/${clusterName}/tools/${installedTool.name}`))
      }
    }

    return (
      <Card 
        className={`flex flex-col h-full ${
          isInstalled && installedTool 
            ? 'cursor-pointer hover:shadow-md hover:border-gray-300 transition-all' 
            : ''
        }`}
        onClick={handleCardClick}
        onKeyDown={handleKeyDown}
        tabIndex={isInstalled && installedTool ? 0 : -1}
        role={isInstalled && installedTool ? 'button' : undefined}
        aria-label={isInstalled && installedTool ? `View details for ${tool.displayName}` : undefined}
      >
      <CardHeader>
        <div className="flex items-start justify-between">
          <div className="flex-1">
            <CardTitle className="text-lg">{tool.displayName}</CardTitle>
            <CardDescription className="text-sm text-muted-foreground mt-1">{toolId}</CardDescription>
            <CardDescription className="mt-1 line-clamp-2">
              {tool.description}
            </CardDescription>
          </div>
        </div>
      </CardHeader>
      <CardContent className="flex flex-col flex-1">
        <div className="space-y-3 flex-1">
          {/* Tool metadata */}
          <div className="flex flex-wrap gap-2">
            <Badge variant="outline" className="text-xs">
              {tool.deploymentMode}
            </Badge>
            {tool.authRequired && (
              <Badge variant="outline" className="text-xs">
                <Shield className="h-3 w-3 mr-1" />
                Auth Required
              </Badge>
            )}
          </div>

          {/* Features */}
          <div className="text-xs text-muted-foreground space-y-1">
            {tool.rbac && (
              <div className="flex items-center gap-1">
                <Shield className="h-3 w-3" />
                <span>RBAC</span>
              </div>
            )}
            {tool.egress && (
              <div className="flex items-center gap-1">
                <Network className="h-3 w-3" />
                <span>Network Restrictions</span>
              </div>
            )}
          </div>
        </div>

        {/* Actions/Status - anchored to bottom */}
        <div className="flex gap-2 pt-3 mt-auto">
            {isInstalled && installedTool ? (
              <>
                <div className="flex-1 flex items-center gap-2">
                  <Badge 
                    variant={['Ready', 'Running'].includes(installedTool.status.phase) ? 'default' : 'secondary'}
                    className={`text-xs ${['Ready', 'Running'].includes(installedTool.status.phase) ? 'bg-green-100 text-green-800 border-green-200' : ''}`}
                  >
                    {['Ready', 'Running'].includes(installedTool.status.phase) ? 'Installed' : installedTool.status.phase}
                  </Badge>
                  {installedTool.status.message && (
                    <span className="text-xs text-muted-foreground truncate">
                      {installedTool.status.message.replace('Image registry is in whitelist', 'approved registry')}
                    </span>
                  )}
                </div>
              </>
            ) : (
              <>
                {isInstalled ? (
                  <Button disabled className="flex-1 bg-green-100 text-green-800 border-green-200 hover:bg-green-100" size="sm">
                    <CheckCircle className="h-4 w-4 mr-2" />
                    Installed
                  </Button>
                ) : (
                  <Button asChild className="flex-1" size="sm">
                    <Link href={getOrgUrl(`/clusters/${clusterName}/tools/install/${toolId}`)}>
                      <Download className="h-4 w-4 mr-2" />
                      Install
                    </Link>
                  </Button>
                )}
                {tool.homepage && (
                  <Button
                    variant="outline"
                    size="sm"
                    asChild
                  >
                    <a href={tool.homepage} target="_blank" rel="noopener noreferrer">
                      <ExternalLink className="h-4 w-4" />
                    </a>
                  </Button>
                )}
              </>
            )}
        </div>
      </CardContent>
    </Card>
    )
  }

  const filteredTools = catalog?.tools
    ? Object.entries(catalog.tools).filter(([_, tool]) => {
        const query = searchQuery.toLowerCase()
        return (
          tool.name.toLowerCase().includes(query) ||
          tool.displayName.toLowerCase().includes(query) ||
          tool.description.toLowerCase().includes(query)
        )
      })
    : []

  // Filter installed tools by search query as well
  const filteredInstalledTools = installedTools.filter((installedTool: InstalledTool) => {
    const catalogEntry = getCatalogEntryForInstalledTool(installedTool)
    if (!catalogEntry) return false
    const query = searchQuery.toLowerCase()
    return (
      catalogEntry.name.toLowerCase().includes(query) ||
      catalogEntry.displayName.toLowerCase().includes(query) ||
      catalogEntry.description.toLowerCase().includes(query) ||
      installedTool.name.toLowerCase().includes(query)
    )
  })

  const isLoading = catalogLoading || toolsLoading
  const error = catalogError || (toolsError ? (toolsError as Error).message : null)

  if (isLoading) {
    return (
        <div className="flex items-center justify-center min-h-[400px]">
          <div className="text-center">
            <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-gray-900 mx-auto"></div>
            <p className="mt-4 text-gray-600">Loading tools...</p>
          </div>
        </div>
    )
  }

  if (error) {
    return (
        <div className="space-y-6">
          <div>
            <h1 className="text-3xl font-bold">Tools</h1>
            <p className="text-gray-600 mt-1">
              Official tools for the {clusterName} cluster
            </p>
          </div>
          <Card>
            <CardContent className="flex flex-col items-center justify-center py-16">
              <Wrench className="h-16 w-16 text-red-500 mb-4" />
              <CardTitle className="text-xl mb-2">Error Loading Tools</CardTitle>
              <CardDescription className="text-center max-w-md">
                {error}
              </CardDescription>
            </CardContent>
          </Card>
        </div>
    )
  }

  return (
      <div className="space-y-6">
        {/* Header */}
        <ResourceHeader
          icon={Wrench}
          title="Tools"
          subtitle="Agents use MCP-compatible tools to perform work"
        />

        {/* Search */}
        <div className="relative">
          <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-gray-400" />
          <Input
            type="text"
            placeholder="Search tools by name or description..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="pl-10"
          />
        </div>

        {/* Installed Tools Section */}
        {filteredInstalledTools.length > 0 && (
          <div>
            <h2 className="text-xl font-semibold mb-4">Installed Tools ({filteredInstalledTools.length})</h2>
            <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
              {filteredInstalledTools.map((installedTool: InstalledTool) => {
                const catalogEntry = getCatalogEntryForInstalledTool(installedTool)
                if (!catalogEntry) return null
                const toolId = installedTool.catalogName || installedTool.name
                return (
                  <ToolCard
                    key={installedTool.name}
                    toolId={toolId}
                    tool={catalogEntry}
                    isInstalled={true}
                    installedTool={installedTool}
                    clusterName={clusterName}
                  />
                )
              })}
            </div>
          </div>
        )}

        {/* Available Tools Section */}
        <div>
          <h2 className="text-xl font-semibold mb-4">Available Tools</h2>
          {filteredTools.length === 0 ? (
            <Card>
              <CardContent className="flex flex-col items-center justify-center py-16">
                <Wrench className="h-16 w-16 text-gray-400 mb-4" />
                <CardTitle className="text-xl mb-2">No tools found</CardTitle>
                <CardDescription className="text-center max-w-md">
                  {searchQuery
                    ? `No tools match your search "${searchQuery}"`
                    : 'No tools available in the catalog'}
                </CardDescription>
              </CardContent>
            </Card>
          ) : (
            <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
              {filteredTools.map(([toolId, tool]) => {
                const installed = isToolInstalled(toolId)
                return (
                  <ToolCard
                    key={toolId}
                    toolId={toolId}
                    tool={tool}
                    isInstalled={installed}
                    clusterName={clusterName}
                  />
                )
              })}
            </div>
          )}
        </div>

        {/* Tool Events */}
        <EventsActivity
          title="Tool Events"
          description="Recent events for tools in this cluster"
          clusterName={clusterName}
          resourceType="tool"
          limit={10}
          showNamespace={false}
        />
      </div>
  )
}