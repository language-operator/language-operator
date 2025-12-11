import { AuthenticatedLayout } from '@/components/layout/authenticated-layout'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Plus, Cloud, AlertCircle, CheckCircle, Globe, Settings, Activity, Users } from 'lucide-react'

export default function ClustersPage() {
  const clusters = [
    {
      id: 'cluster-1',
      name: 'production',
      description: 'Main production cluster serving customer requests',
      status: 'Running',
      url: 'https://api.langop.com',
      replicas: 3,
      agents: ['customer-support-v2', 'content-writer', 'technical-assistant'],
      requests: '125.3k',
      uptime: '99.98%',
      lastDeploy: '2 days ago',
      namespace: 'production',
      resources: {
        cpu: '2.1/4 cores',
        memory: '3.8/8 GB',
      },
      health: 'Healthy',
    },
    {
      id: 'cluster-2',
      name: 'staging',
      description: 'Staging environment for testing new configurations',
      status: 'Running',
      url: 'https://staging.langop.com',
      replicas: 2,
      agents: ['customer-support-v3', 'code-reviewer-beta'],
      requests: '8.7k',
      uptime: '98.45%',
      lastDeploy: '6 hours ago',
      namespace: 'staging',
      resources: {
        cpu: '1.2/2 cores',
        memory: '1.8/4 GB',
      },
      health: 'Healthy',
    },
    {
      id: 'cluster-3',
      name: 'development',
      description: 'Development cluster for testing experimental features',
      status: 'Stopped',
      url: 'https://dev.langop.com',
      replicas: 1,
      agents: ['test-agent'],
      requests: '234',
      uptime: '85.12%',
      lastDeploy: '1 week ago',
      namespace: 'development',
      resources: {
        cpu: '0.1/1 core',
        memory: '0.5/2 GB',
      },
      health: 'Stopped',
    },
    {
      id: 'cluster-4',
      name: 'analytics',
      description: 'Specialized cluster for data analysis and reporting',
      status: 'Error',
      url: 'https://analytics.langop.com',
      replicas: 2,
      agents: ['data-analyzer', 'report-generator'],
      requests: '15.2k',
      uptime: '94.23%',
      lastDeploy: '3 days ago',
      namespace: 'analytics',
      resources: {
        cpu: '3.8/4 cores',
        memory: '7.2/8 GB',
      },
      health: 'Degraded',
    },
  ]

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'Running':
        return <CheckCircle className="h-4 w-4 text-green-500" />
      case 'Stopped':
        return <AlertCircle className="h-4 w-4 text-gray-500" />
      case 'Error':
        return <AlertCircle className="h-4 w-4 text-red-500" />
      default:
        return <AlertCircle className="h-4 w-4 text-gray-500" />
    }
  }

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'Running':
        return 'bg-green-100 text-green-800'
      case 'Stopped':
        return 'bg-gray-100 text-gray-800'
      case 'Error':
        return 'bg-red-100 text-red-800'
      default:
        return 'bg-gray-100 text-gray-800'
    }
  }

  const getHealthColor = (health: string) => {
    switch (health) {
      case 'Healthy':
        return 'text-green-600'
      case 'Degraded':
        return 'text-yellow-600'
      case 'Stopped':
        return 'text-gray-600'
      default:
        return 'text-red-600'
    }
  }

  return (
    <AuthenticatedLayout>
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold">Language Clusters</h1>
            <p className="text-gray-600 mt-2">
              Manage deployments and HTTP endpoints for your agents
            </p>
          </div>
          <Button>
            <Plus className="h-4 w-4 mr-2" />
            Deploy Cluster
          </Button>
        </div>

        {/* Stats */}
        <div className="grid gap-4 md:grid-cols-4">
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Total Clusters</CardTitle>
              <Cloud className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{clusters.length}</div>
            </CardContent>
          </Card>
          
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Running</CardTitle>
              <CheckCircle className="h-4 w-4 text-green-500" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">
                {clusters.filter(c => c.status === 'Running').length}
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Total Requests</CardTitle>
              <Activity className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">149.5k</div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Avg Uptime</CardTitle>
              <Settings className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">96.9%</div>
            </CardContent>
          </Card>
        </div>

        {/* Clusters List */}
        <div className="space-y-4">
          {clusters.map((cluster) => (
            <Card key={cluster.id}>
              <CardHeader>
                <div className="flex items-start justify-between">
                  <div className="flex items-start space-x-4">
                    <Cloud className="h-6 w-6 text-orange-500 mt-1" />
                    <div>
                      <div className="flex items-center space-x-2">
                        <CardTitle className="text-lg">{cluster.name}</CardTitle>
                        <Badge variant="secondary" className="text-xs">
                          {cluster.namespace}
                        </Badge>
                        <Badge variant="outline" className="text-xs">
                          {cluster.replicas} replica{cluster.replicas !== 1 ? 's' : ''}
                        </Badge>
                      </div>
                      <CardDescription className="mt-1 max-w-2xl">
                        {cluster.description}
                      </CardDescription>
                      <div className="flex items-center space-x-4 mt-2 text-sm text-gray-600">
                        <div className="flex items-center space-x-1">
                          <Globe className="h-4 w-4" />
                          <span>{cluster.url}</span>
                        </div>
                        <div className="flex items-center space-x-1">
                          <Activity className="h-4 w-4" />
                          <span className={getHealthColor(cluster.health)}>
                            {cluster.health}
                          </span>
                        </div>
                      </div>
                    </div>
                  </div>
                  <div className="flex items-center space-x-2">
                    {getStatusIcon(cluster.status)}
                    <Badge className={getStatusColor(cluster.status)}>
                      {cluster.status}
                    </Badge>
                  </div>
                </div>
              </CardHeader>
              <CardContent>
                <div className="grid gap-4 md:grid-cols-5">
                  <div>
                    <h4 className="text-sm font-medium text-gray-600 mb-1">Agents</h4>
                    <div className="space-y-1">
                      {cluster.agents.slice(0, 2).map((agent) => (
                        <div key={agent} className="text-sm">{agent}</div>
                      ))}
                      {cluster.agents.length > 2 && (
                        <div className="text-sm text-gray-500">
                          +{cluster.agents.length - 2} more
                        </div>
                      )}
                    </div>
                  </div>
                  
                  <div>
                    <h4 className="text-sm font-medium text-gray-600 mb-1">Resources</h4>
                    <div className="text-sm">
                      <div>CPU: {cluster.resources.cpu}</div>
                      <div>RAM: {cluster.resources.memory}</div>
                    </div>
                  </div>

                  <div>
                    <h4 className="text-sm font-medium text-gray-600 mb-1">Performance</h4>
                    <div className="text-sm">
                      <div>Requests: {cluster.requests}</div>
                      <div>Uptime: {cluster.uptime}</div>
                    </div>
                  </div>

                  <div>
                    <h4 className="text-sm font-medium text-gray-600 mb-1">Last Deploy</h4>
                    <p className="text-sm">{cluster.lastDeploy}</p>
                  </div>

                  <div>
                    <h4 className="text-sm font-medium text-gray-600 mb-1">Actions</h4>
                    <div className="flex space-x-1">
                      <Button variant="outline" size="sm">
                        Edit
                      </Button>
                      {cluster.status === 'Running' ? (
                        <Button variant="outline" size="sm">
                          Stop
                        </Button>
                      ) : (
                        <Button variant="outline" size="sm">
                          Start
                        </Button>
                      )}
                      <Button variant="outline" size="sm">
                        Logs
                      </Button>
                    </div>
                  </div>
                </div>

                {/* Additional cluster details */}
                <div className="mt-4 pt-4 border-t">
                  <div className="flex items-center justify-between text-sm">
                    <div className="flex items-center space-x-6">
                      <div className="flex items-center space-x-1">
                        <Users className="h-4 w-4 text-gray-400" />
                        <span className="text-gray-600">{cluster.agents.length} agents</span>
                      </div>
                      <div className="flex items-center space-x-1">
                        <Activity className="h-4 w-4 text-gray-400" />
                        <span className="text-gray-600">{cluster.requests} requests</span>
                      </div>
                      <div className="flex items-center space-x-1">
                        <Settings className="h-4 w-4 text-gray-400" />
                        <span className="text-gray-600">Updated {cluster.lastDeploy}</span>
                      </div>
                    </div>
                    <Button variant="ghost" size="sm">
                      View Details
                    </Button>
                  </div>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      </div>
    </AuthenticatedLayout>
  )
}