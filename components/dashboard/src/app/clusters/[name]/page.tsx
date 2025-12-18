'use client'

import { useParams } from 'next/navigation'
import { AuthenticatedLayout } from '@/components/layout/authenticated-layout'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { ResourceHeader } from '@/components/ui/resource-header'
import { ClusterStatusBadge } from '@/components/ui/resource-status-badge'
import { 
  Bot, 
  Cpu, 
  Wrench, 
  Users, 
  Activity,
  ExternalLink,
  Edit,
  Settings,
  BarChart3
} from 'lucide-react'
import { useCluster } from '@/hooks/use-clusters'
import { useResourceCounts } from '@/hooks/useResourceCounts'
import Link from 'next/link'

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

export default function ClusterDashboard() {
  const params = useParams()
  const clusterName = params?.name as string
  
  const { data: cluster, isLoading, error } = useCluster(clusterName)
  const { counts, loading: countsLoading } = useResourceCounts(clusterName)

  if (isLoading) {
    return (
      <AuthenticatedLayout>
        <div className="flex items-center justify-center h-64">
          <div className="text-lg">Loading cluster dashboard...</div>
        </div>
      </AuthenticatedLayout>
    )
  }

  if (error || !cluster) {
    return (
      <AuthenticatedLayout>
        <div className="flex items-center justify-center h-64">
          <div className="text-center">
            <div className="text-lg text-red-600">Cluster not found</div>
            <div className="text-sm text-gray-500 mt-2">
              The cluster "{clusterName}" could not be loaded.
            </div>
          </div>
        </div>
      </AuthenticatedLayout>
    )
  }


  return (
    <AuthenticatedLayout>
      <div className="space-y-6">
        {/* Cluster Header */}
        <ResourceHeader
          icon={BarChart3}
          iconColor="text-blue-500"
          title={cluster.metadata?.name || ''}
          subtitle="LanguageCluster"
          actions={
            <Button variant="outline" size="sm" asChild>
              <Link href={`/clusters/${clusterName}/edit`}>
                <Settings className="h-4 w-4 mr-2" />
                Settings
              </Link>
            </Button>
          }
        />

        {/* Cluster Info */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          <Card>
            <CardHeader>
              <CardTitle>Cluster Information</CardTitle>
              <CardDescription>
                Basic details about this language cluster
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div>
                <dt className="text-sm font-medium text-gray-500">Name</dt>
                <dd className="mt-1 text-sm text-gray-900">{cluster.metadata?.name}</dd>
              </div>
              {cluster.spec?.domain && (
                <div>
                  <dt className="text-sm font-medium text-gray-500">Domain</dt>
                  <dd className="mt-1 text-sm text-gray-900">
                    <a 
                      href={`https://${cluster.spec.domain}`} 
                      target="_blank" 
                      rel="noopener noreferrer"
                      className="flex items-center gap-1 text-blue-600 hover:text-blue-800"
                    >
                      {cluster.spec.domain}
                      <ExternalLink className="h-3 w-3" />
                    </a>
                  </dd>
                </div>
              )}
              <div>
                <dt className="text-sm font-medium text-gray-500">Namespace</dt>
                <dd className="mt-1 text-sm text-gray-900">{cluster.metadata?.namespace}</dd>
              </div>
              <div>
                <dt className="text-sm font-medium text-gray-500">Status</dt>
                <dd className="mt-1">
                  <ClusterStatusBadge cluster={cluster} />
                </dd>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Quick Actions</CardTitle>
              <CardDescription>
                Common tasks for managing this cluster
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="grid grid-cols-2 gap-4">
                <Link href={`/clusters/${clusterName}/models/new`}>
                  <div className="flex flex-col items-center p-4 border rounded-lg hover:bg-gray-50 cursor-pointer transition-colors">
                    <Cpu className="h-8 w-8 text-green-500 mb-2" />
                    <span className="text-sm font-medium">Create Model</span>
                    <span className="text-xs text-gray-500 text-center mt-1">
                      Connect to LLM provider
                    </span>
                  </div>
                </Link>
                <Link href={`/clusters/${clusterName}/tools`}>
                  <div className="flex flex-col items-center p-4 border rounded-lg hover:bg-gray-50 cursor-pointer transition-colors">
                    <Wrench className="h-8 w-8 text-purple-500 mb-2" />
                    <span className="text-sm font-medium">Install Tool</span>
                    <span className="text-xs text-gray-500 text-center mt-1">
                      Add capabilities to agents
                    </span>
                  </div>
                </Link>
                <Link href={`/clusters/${clusterName}/personas/new`}>
                  <div className="flex flex-col items-center p-4 border rounded-lg hover:bg-gray-50 cursor-pointer transition-colors">
                    <Users className="h-8 w-8 text-rose-500 mb-2" />
                    <span className="text-sm font-medium">Create Persona</span>
                    <span className="text-xs text-gray-500 text-center mt-1">
                      Define agent behavior
                    </span>
                  </div>
                </Link>
                <Link href={`/clusters/${clusterName}/agents/new`}>
                  <div className="flex flex-col items-center p-4 border rounded-lg hover:bg-gray-50 cursor-pointer transition-colors">
                    <Bot className="h-8 w-8 text-blue-500 mb-2" />
                    <span className="text-sm font-medium">Create Agent</span>
                    <span className="text-xs text-gray-500 text-center mt-1">
                      Build a new AI agent
                    </span>
                  </div>
                </Link>
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Resource Overview Cards */}
        <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
          <Link href={`/clusters/${clusterName}/models`}>
            <Card className="hover:shadow-md transition-shadow cursor-pointer">
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">Models</CardTitle>
                <Cpu className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">
                  {countsLoading ? '-' : counts?.models || 0}
                </div>
                <p className="text-xs text-muted-foreground">
                  LanguageModels
                </p>
              </CardContent>
            </Card>
          </Link>
          <Link href={`/clusters/${clusterName}/tools`}>
            <Card className="hover:shadow-md transition-shadow cursor-pointer">
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">Tools</CardTitle>
                <Wrench className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">
                  {countsLoading ? '-' : counts?.tools || 0}
                </div>
                <p className="text-xs text-muted-foreground">
                  LanguageTools
                </p>
              </CardContent>
            </Card>
          </Link>
          <Link href={`/clusters/${clusterName}/personas`}>
            <Card className="hover:shadow-md transition-shadow cursor-pointer">
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">Personas</CardTitle>
                <Users className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">
                  {countsLoading ? '-' : counts?.personas || 0}
                </div>
                <p className="text-xs text-muted-foreground">
                  LanguagePersonas
                </p>
              </CardContent>
            </Card>
          </Link>
          <Link href={`/clusters/${clusterName}/agents`}>
            <Card className="hover:shadow-md transition-shadow cursor-pointer">
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">Agents</CardTitle>
                <Bot className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">
                  {countsLoading ? '-' : counts?.agents || 0}
                </div>
                <p className="text-xs text-muted-foreground">
                  LanguageAgents
                </p>
              </CardContent>
            </Card>
          </Link>
        </div>
      </div>
    </AuthenticatedLayout>
  )
}