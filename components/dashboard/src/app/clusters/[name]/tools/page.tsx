'use client'

import { useParams, useRouter } from 'next/navigation'
import { AuthenticatedLayout } from '@/components/layout/authenticated-layout'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Wrench, Download, CheckCircle, Search, ExternalLink, Shield, Network } from 'lucide-react'
import Link from 'next/link'
import { useEffect, useState } from 'react'
import { ToolCatalog, ToolCatalogEntry, InstalledTool } from '@/types/tool-catalog'

export default function ClusterTools() {
  const params = useParams()
  const clusterName = params?.name as string
  const [catalog, setCatalog] = useState<ToolCatalog | null>(null)
  const [installedTools, setInstalledTools] = useState<InstalledTool[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [searchQuery, setSearchQuery] = useState('')

  useEffect(() => {
    fetchData()
  }, [clusterName])

  const fetchData = async () => {
    try {
      setLoading(true)
      setError(null)

      // Fetch catalog
      const catalogResponse = await fetch('/api/tools/catalog')
      if (!catalogResponse.ok) {
        throw new Error('Failed to fetch tool catalog')
      }
      const catalogData = await catalogResponse.json()
      setCatalog(catalogData)

      // Fetch installed tools (API will determine namespace from session)
      const toolsResponse = await fetch(`/api/tools`)
      if (toolsResponse.ok) {
        const toolsData = await toolsResponse.json()
        // Convert LanguageTool objects to InstalledTool format
        const adaptedTools = (toolsData.data || []).map((tool: any) => ({
          name: tool.metadata.name,
          catalogName: tool.metadata.labels?.['langop.io/catalog-name'] || tool.metadata.name,
          status: {
            phase: tool.status?.phase || 'Unknown',
            message: tool.status?.conditions?.[0]?.message || ''
          }
        }))
        setInstalledTools(adaptedTools)
      }
    } catch (err) {
      console.error('Error fetching data:', err)
      setError(err instanceof Error ? err.message : 'Failed to load tools')
    } finally {
      setLoading(false)
    }
  }

  const isToolInstalled = (toolName: string) => {
    return installedTools.some(tool => 
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
      
      router.push(`/clusters/${clusterName}/tools/${installedTool.name}`)
    }

    const handleKeyDown = (event: React.KeyboardEvent) => {
      if (!isInstalled || !installedTool) return
      
      if (event.key === 'Enter' || event.key === ' ') {
        event.preventDefault()
        router.push(`/clusters/${clusterName}/tools/${installedTool.name}`)
      }
    }

    return (
      <Card 
        key={toolId} 
        className={`flex flex-col h-full ${
          isInstalled && installedTool 
            ? 'border-green-200 bg-green-50/50 cursor-pointer hover:bg-green-50/70 hover:border-green-300 transition-colors' 
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
            <CardDescription className="text-sm text-gray-500 mt-1">{toolId}</CardDescription>
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
            <Badge variant="secondary" className="text-xs">
              {tool.type.toUpperCase()}
            </Badge>
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
          <div className="text-xs text-gray-600 space-y-1">
            {tool.rbac && (
              <div className="flex items-center gap-1">
                <Shield className="h-3 w-3" />
                <span>RBAC configured</span>
              </div>
            )}
            {tool.egress && (
              <div className="flex items-center gap-1">
                <Network className="h-3 w-3" />
                <span>Network policies defined</span>
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
                    variant={installedTool.status.phase === 'Ready' ? 'default' : 'secondary'}
                    className="text-xs"
                  >
                    {installedTool.status.phase}
                  </Badge>
                  {installedTool.status.message && (
                    <span className="text-xs text-gray-500 truncate">
                      {installedTool.status.message}
                    </span>
                  )}
                </div>
                <Button 
                  variant="outline" 
                  size="sm" 
                  disabled
                  onClick={(e) => e.stopPropagation()}
                >
                  Configure
                </Button>
                <Button 
                  variant="destructive" 
                  size="sm" 
                  disabled
                  onClick={(e) => e.stopPropagation()}
                >
                  Remove
                </Button>
              </>
            ) : (
              <>
                {isInstalled ? (
                  <Button disabled className="flex-1" size="sm">
                    <CheckCircle className="h-4 w-4 mr-2" />
                    Installed
                  </Button>
                ) : (
                  <Button asChild className="flex-1" size="sm">
                    <Link href={`/clusters/${clusterName}/tools/install/${toolId}`}>
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
  const filteredInstalledTools = installedTools.filter(installedTool => {
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

  if (loading) {
    return (
      <AuthenticatedLayout>
        <div className="flex items-center justify-center min-h-[400px]">
          <div className="text-center">
            <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-gray-900 mx-auto"></div>
            <p className="mt-4 text-gray-600">Loading tool catalog...</p>
          </div>
        </div>
      </AuthenticatedLayout>
    )
  }

  if (error) {
    return (
      <AuthenticatedLayout>
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
              <Button onClick={fetchData} className="mt-4">
                Try Again
              </Button>
            </CardContent>
          </Card>
        </div>
      </AuthenticatedLayout>
    )
  }

  return (
    <AuthenticatedLayout>
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold">Tools Catalog</h1>
            <p className="text-gray-600 mt-1">
              Browse and install official tools for the {clusterName} cluster
            </p>
          </div>
        </div>

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
            <h2 className="text-xl font-semibold mb-4">Installed Tools</h2>
            <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
              {filteredInstalledTools.map((installedTool) => {
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
      </div>
    </AuthenticatedLayout>
  )
}