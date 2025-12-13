'use client'

import { useParams } from 'next/navigation'
import { AuthenticatedLayout } from '@/components/layout/authenticated-layout'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { 
  Bot, 
  Cpu, 
  Wrench, 
  Users, 
  Activity,
  ExternalLink,
  Edit,
  Settings
} from 'lucide-react'
import { useCluster } from '@/hooks/use-clusters'
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

  const getStatusColor = (phase?: string) => {
    switch (phase) {
      case 'Ready':
        return 'bg-green-100 text-green-800'
      case 'Pending':
        return 'bg-yellow-100 text-yellow-800'
      case 'Failed':
        return 'bg-red-100 text-red-800'
      default:
        return 'bg-gray-100 text-gray-800'
    }
  }

  return (
    <AuthenticatedLayout>
      <div className="space-y-6">
        {/* Cluster Header */}
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold">{cluster.metadata?.name}</h1>
            <p className="text-gray-600 mt-1">
              Created {formatTimeAgo(cluster.metadata?.creationTimestamp)}
            </p>
          </div>
          <div className="flex items-center gap-4">
            <Badge className={getStatusColor(cluster.status?.phase)}>
              <Activity className="h-3 w-3 mr-1" />
              {cluster.status?.phase || 'Unknown'}
            </Badge>
            <Button variant="outline" size="sm" asChild>
              <Link href={`/clusters/${clusterName}/settings`}>
                <Settings className="h-4 w-4 mr-2" />
                Settings
              </Link>
            </Button>
          </div>
        </div>

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
                  <Badge className={getStatusColor(cluster.status?.phase)}>
                    {cluster.status?.phase || 'Unknown'}
                  </Badge>
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
                <Button variant="outline" className="h-20 flex-col" asChild>
                  <Link href={`/clusters/${clusterName}/models`}>
                    <Cpu className="h-6 w-6 mb-2" />
                    Models
                  </Link>
                </Button>
                <Button variant="outline" className="h-20 flex-col" asChild>
                  <Link href={`/clusters/${clusterName}/tools`}>
                    <Wrench className="h-6 w-6 mb-2" />
                    Tools
                  </Link>
                </Button>
                <Button variant="outline" className="h-20 flex-col" asChild>
                  <Link href={`/clusters/${clusterName}/personas`}>
                    <Users className="h-6 w-6 mb-2" />
                    Personas
                  </Link>
                </Button>
                <Button variant="outline" className="h-20 flex-col" asChild>
                  <Link href={`/clusters/${clusterName}/agents`}>
                    <Bot className="h-6 w-6 mb-2" />
                    Agents
                  </Link>
                </Button>
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Resource Overview Cards */}
        <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Models</CardTitle>
              <Cpu className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">0</div>
              <p className="text-xs text-muted-foreground">
                Language models in this cluster
              </p>
            </CardContent>
          </Card>
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Tools</CardTitle>
              <Wrench className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">0</div>
              <p className="text-xs text-muted-foreground">
                Available tools
              </p>
            </CardContent>
          </Card>
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Personas</CardTitle>
              <Users className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">0</div>
              <p className="text-xs text-muted-foreground">
                Configured personas
              </p>
            </CardContent>
          </Card>
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Agents</CardTitle>
              <Bot className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">0</div>
              <p className="text-xs text-muted-foreground">
                Running agents
              </p>
            </CardContent>
          </Card>
        </div>
      </div>
    </AuthenticatedLayout>
  )
}