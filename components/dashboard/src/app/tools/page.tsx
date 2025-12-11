import { AuthenticatedLayout } from '@/components/layout/authenticated-layout'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Plus, Wrench, AlertCircle, CheckCircle, Clock, Globe, Database, Mail, Code } from 'lucide-react'

export default function ToolsPage() {
  const tools = [
    {
      id: 'tool-1',
      name: 'web-search',
      description: 'Search the web for current information and answers',
      status: 'Active',
      category: 'Search',
      icon: Globe,
      version: 'v1.2.0',
      lastUsed: '2 minutes ago',
      usageCount: '15.3k',
      parameters: ['query', 'max_results', 'safe_search'],
      namespace: 'production',
    },
    {
      id: 'tool-2',
      name: 'knowledge-base',
      description: 'Query internal knowledge base and documentation',
      status: 'Active',
      category: 'Data',
      icon: Database,
      version: 'v2.1.0',
      lastUsed: '5 minutes ago',
      usageCount: '8.7k',
      parameters: ['query', 'collection', 'limit'],
      namespace: 'production',
    },
    {
      id: 'tool-3',
      name: 'email-sender',
      description: 'Send emails with templates and attachments',
      status: 'Pending',
      category: 'Communication',
      icon: Mail,
      version: 'v1.0.0',
      lastUsed: '1 hour ago',
      usageCount: '2.1k',
      parameters: ['recipient', 'subject', 'template', 'attachments'],
      namespace: 'staging',
    },
    {
      id: 'tool-4',
      name: 'code-analysis',
      description: 'Analyze code for bugs, security issues, and improvements',
      status: 'Error',
      category: 'Development',
      icon: Code,
      version: 'v3.0.1',
      lastUsed: '2 days ago',
      usageCount: '945',
      parameters: ['code', 'language', 'analysis_type'],
      namespace: 'development',
    },
    {
      id: 'tool-5',
      name: 'git-operations',
      description: 'Perform Git operations like commits, branches, and merges',
      status: 'Active',
      category: 'Development',
      icon: Code,
      version: 'v1.5.0',
      lastUsed: '30 minutes ago',
      usageCount: '3.2k',
      parameters: ['repository', 'operation', 'branch', 'message'],
      namespace: 'development',
    },
  ]

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'Active':
        return <CheckCircle className="h-4 w-4 text-green-500" />
      case 'Pending':
        return <Clock className="h-4 w-4 text-yellow-500" />
      case 'Error':
        return <AlertCircle className="h-4 w-4 text-red-500" />
      default:
        return <AlertCircle className="h-4 w-4 text-gray-500" />
    }
  }

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'Active':
        return 'bg-green-100 text-green-800'
      case 'Pending':
        return 'bg-yellow-100 text-yellow-800'
      case 'Error':
        return 'bg-red-100 text-red-800'
      default:
        return 'bg-gray-100 text-gray-800'
    }
  }

  const getCategoryColor = (category: string) => {
    switch (category) {
      case 'Search':
        return 'bg-blue-100 text-blue-800'
      case 'Data':
        return 'bg-green-100 text-green-800'
      case 'Communication':
        return 'bg-purple-100 text-purple-800'
      case 'Development':
        return 'bg-orange-100 text-orange-800'
      default:
        return 'bg-gray-100 text-gray-800'
    }
  }

  return (
    <AuthenticatedLayout>
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold">Language Tools</h1>
            <p className="text-gray-600 mt-2">
              Manage tools and capabilities available to your agents
            </p>
          </div>
          <Button>
            <Plus className="h-4 w-4 mr-2" />
            Add Tool
          </Button>
        </div>

        {/* Stats */}
        <div className="grid gap-4 md:grid-cols-4">
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Total Tools</CardTitle>
              <Wrench className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{tools.length}</div>
            </CardContent>
          </Card>
          
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Active</CardTitle>
              <CheckCircle className="h-4 w-4 text-green-500" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">
                {tools.filter(t => t.status === 'Active').length}
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Categories</CardTitle>
              <Wrench className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">
                {new Set(tools.map(t => t.category)).size}
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Total Usage</CardTitle>
              <AlertCircle className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">30.2k</div>
            </CardContent>
          </Card>
        </div>

        {/* Tools List */}
        <div className="space-y-4">
          {tools.map((tool) => {
            const IconComponent = tool.icon
            return (
              <Card key={tool.id}>
                <CardHeader>
                  <div className="flex items-start justify-between">
                    <div className="flex items-start space-x-4">
                      <IconComponent className="h-6 w-6 text-purple-500 mt-1" />
                      <div>
                        <div className="flex items-center space-x-2">
                          <CardTitle className="text-lg">{tool.name}</CardTitle>
                          <Badge className={getCategoryColor(tool.category)}>
                            {tool.category}
                          </Badge>
                          <Badge variant="secondary" className="text-xs">
                            {tool.namespace}
                          </Badge>
                        </div>
                        <CardDescription className="mt-1">
                          {tool.description}
                        </CardDescription>
                      </div>
                    </div>
                    <div className="flex items-center space-x-2">
                      {getStatusIcon(tool.status)}
                      <Badge className={getStatusColor(tool.status)}>
                        {tool.status}
                      </Badge>
                    </div>
                  </div>
                </CardHeader>
                <CardContent>
                  <div className="grid gap-4 md:grid-cols-4">
                    <div>
                      <h4 className="text-sm font-medium text-gray-600 mb-1">Version</h4>
                      <p className="text-sm">{tool.version}</p>
                    </div>
                    <div>
                      <h4 className="text-sm font-medium text-gray-600 mb-1">Usage Count</h4>
                      <p className="text-sm">{tool.usageCount}</p>
                    </div>
                    <div>
                      <h4 className="text-sm font-medium text-gray-600 mb-1">Last Used</h4>
                      <p className="text-sm">{tool.lastUsed}</p>
                    </div>
                    <div>
                      <h4 className="text-sm font-medium text-gray-600 mb-1">Parameters</h4>
                      <div className="flex flex-wrap gap-1">
                        {tool.parameters.slice(0, 3).map((param) => (
                          <Badge key={param} variant="outline" className="text-xs">
                            {param}
                          </Badge>
                        ))}
                        {tool.parameters.length > 3 && (
                          <Badge variant="outline" className="text-xs">
                            +{tool.parameters.length - 3} more
                          </Badge>
                        )}
                      </div>
                    </div>
                  </div>
                  <div className="flex space-x-2 mt-4">
                    <Button variant="outline" size="sm">
                      Edit
                    </Button>
                    <Button variant="outline" size="sm">
                      Test
                    </Button>
                    <Button variant="outline" size="sm">
                      View Logs
                    </Button>
                    <Button variant="destructive" size="sm">
                      Delete
                    </Button>
                  </div>
                </CardContent>
              </Card>
            )
          })}
        </div>
      </div>
    </AuthenticatedLayout>
  )
}