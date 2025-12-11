import { AuthenticatedLayout } from '@/components/layout/authenticated-layout'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Plus, Cpu, AlertCircle, CheckCircle, Settings } from 'lucide-react'

export default function ModelsPage() {
  const models = [
    {
      id: 'model-1',
      name: 'gpt-4-turbo',
      provider: 'OpenAI',
      status: 'Active',
      version: 'gpt-4-0125-preview',
      contextWindow: '128k tokens',
      cost: '$0.01/1k tokens',
      lastUsed: '5 minutes ago',
      totalRequests: '1.2M',
      namespace: 'production',
    },
    {
      id: 'model-2',
      name: 'claude-3-sonnet',
      provider: 'Anthropic',
      status: 'Active',
      version: 'claude-3-sonnet-20240229',
      contextWindow: '200k tokens',
      cost: '$0.015/1k tokens',
      lastUsed: '2 hours ago',
      totalRequests: '850k',
      namespace: 'production',
    },
    {
      id: 'model-3',
      name: 'llama-2-70b',
      provider: 'Meta',
      status: 'Inactive',
      version: 'llama-2-70b-chat',
      contextWindow: '4k tokens',
      cost: '$0.0008/1k tokens',
      lastUsed: '2 weeks ago',
      totalRequests: '45k',
      namespace: 'development',
    },
    {
      id: 'model-4',
      name: 'gpt-3.5-turbo',
      provider: 'OpenAI',
      status: 'Error',
      version: 'gpt-3.5-turbo-0125',
      contextWindow: '16k tokens',
      cost: '$0.0005/1k tokens',
      lastUsed: '1 day ago',
      totalRequests: '2.8M',
      namespace: 'staging',
    },
  ]

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'Active':
        return <CheckCircle className="h-4 w-4 text-green-500" />
      case 'Inactive':
        return <Settings className="h-4 w-4 text-gray-500" />
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
      case 'Inactive':
        return 'bg-gray-100 text-gray-800'
      case 'Error':
        return 'bg-red-100 text-red-800'
      default:
        return 'bg-gray-100 text-gray-800'
    }
  }

  const getProviderColor = (provider: string) => {
    switch (provider) {
      case 'OpenAI':
        return 'bg-blue-100 text-blue-800'
      case 'Anthropic':
        return 'bg-purple-100 text-purple-800'
      case 'Meta':
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
            <h1 className="text-3xl font-bold">Language Models</h1>
            <p className="text-gray-600 mt-2">
              Configure and manage your LLM provider connections
            </p>
          </div>
          <Button>
            <Plus className="h-4 w-4 mr-2" />
            Add Model
          </Button>
        </div>

        {/* Stats */}
        <div className="grid gap-4 md:grid-cols-4">
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Total Models</CardTitle>
              <Cpu className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{models.length}</div>
            </CardContent>
          </Card>
          
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Active</CardTitle>
              <CheckCircle className="h-4 w-4 text-green-500" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">
                {models.filter(m => m.status === 'Active').length}
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Providers</CardTitle>
              <Settings className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">
                {new Set(models.map(m => m.provider)).size}
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Total Requests</CardTitle>
              <AlertCircle className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">5.1M</div>
            </CardContent>
          </Card>
        </div>

        {/* Models List */}
        <div className="space-y-4">
          {models.map((model) => (
            <Card key={model.id}>
              <CardHeader>
                <div className="flex items-start justify-between">
                  <div className="flex items-start space-x-4">
                    <Cpu className="h-6 w-6 text-green-500 mt-1" />
                    <div>
                      <div className="flex items-center space-x-2">
                        <CardTitle className="text-lg">{model.name}</CardTitle>
                        <Badge className={getProviderColor(model.provider)}>
                          {model.provider}
                        </Badge>
                        <Badge variant="secondary" className="text-xs">
                          {model.namespace}
                        </Badge>
                      </div>
                      <CardDescription className="mt-1">
                        {model.version}
                      </CardDescription>
                    </div>
                  </div>
                  <div className="flex items-center space-x-2">
                    {getStatusIcon(model.status)}
                    <Badge className={getStatusColor(model.status)}>
                      {model.status}
                    </Badge>
                  </div>
                </div>
              </CardHeader>
              <CardContent>
                <div className="grid gap-4 md:grid-cols-5">
                  <div>
                    <h4 className="text-sm font-medium text-gray-600 mb-1">Context Window</h4>
                    <p className="text-sm">{model.contextWindow}</p>
                  </div>
                  <div>
                    <h4 className="text-sm font-medium text-gray-600 mb-1">Cost</h4>
                    <p className="text-sm">{model.cost}</p>
                  </div>
                  <div>
                    <h4 className="text-sm font-medium text-gray-600 mb-1">Total Requests</h4>
                    <p className="text-sm">{model.totalRequests}</p>
                  </div>
                  <div>
                    <h4 className="text-sm font-medium text-gray-600 mb-1">Last Used</h4>
                    <p className="text-sm">{model.lastUsed}</p>
                  </div>
                  <div>
                    <h4 className="text-sm font-medium text-gray-600 mb-1">Actions</h4>
                    <div className="flex space-x-2">
                      <Button variant="outline" size="sm">
                        Edit
                      </Button>
                      <Button variant="outline" size="sm">
                        Test
                      </Button>
                      <Button variant="destructive" size="sm">
                        Delete
                      </Button>
                    </div>
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