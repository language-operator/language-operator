'use client'

import { AuthenticatedLayout } from '@/components/layout/authenticated-layout'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Plus, Bot, AlertCircle, CheckCircle, Clock, Loader2 } from 'lucide-react'
import { useAgents, useDeleteAgent } from '@/hooks/use-agents'
import { LanguageAgent } from '@/lib/kubernetes'

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

export default function AgentsPage() {
  const { data: agents = [], isLoading, error } = useAgents()
  const deleteAgent = useDeleteAgent()

  const getStatusIcon = (agent: LanguageAgent) => {
    const phase = agent.status?.phase || 'Unknown'
    const ready = agent.status?.ready
    
    if (ready) {
      return <CheckCircle className="h-4 w-4 text-green-500" />
    } else if (phase === 'Pending' || phase === 'Creating') {
      return <Clock className="h-4 w-4 text-yellow-500" />
    } else {
      return <AlertCircle className="h-4 w-4 text-red-500" />
    }
  }

  const getStatusColor = (agent: LanguageAgent) => {
    const phase = agent.status?.phase || 'Unknown'
    const ready = agent.status?.ready
    
    if (ready) {
      return 'bg-green-100 text-green-800'
    } else if (phase === 'Pending' || phase === 'Creating') {
      return 'bg-yellow-100 text-yellow-800'
    } else {
      return 'bg-red-100 text-red-800'
    }
  }

  const getDisplayStatus = (agent: LanguageAgent) => {
    return agent.status?.ready ? 'Ready' : (agent.status?.phase || 'Unknown')
  }

  const handleDeleteAgent = async (agentName: string) => {
    if (confirm(`Are you sure you want to delete agent "${agentName}"?`)) {
      try {
        await deleteAgent.mutateAsync(agentName)
      } catch (error) {
        console.error('Failed to delete agent:', error)
        alert('Failed to delete agent. Please try again.')
      }
    }
  }

  if (isLoading) {
    return (
      <AuthenticatedLayout>
        <div className="flex items-center justify-center h-64">
          <Loader2 className="h-8 w-8 animate-spin" />
          <span className="ml-2">Loading agents...</span>
        </div>
      </AuthenticatedLayout>
    )
  }

  if (error) {
    return (
      <AuthenticatedLayout>
        <div className="flex items-center justify-center h-64">
          <AlertCircle className="h-8 w-8 text-red-500" />
          <span className="ml-2">Failed to load agents</span>
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
            <h1 className="text-3xl font-bold">Language Agents</h1>
            <p className="text-gray-600 mt-2">
              Manage your AI agents and their configurations
            </p>
          </div>
          <Button>
            <Plus className="h-4 w-4 mr-2" />
            Create Agent
          </Button>
        </div>

        {/* Stats */}
        <div className="grid gap-4 md:grid-cols-4">
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Total Agents</CardTitle>
              <Bot className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{agents.length}</div>
            </CardContent>
          </Card>
          
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Ready</CardTitle>
              <CheckCircle className="h-4 w-4 text-green-500" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">
                {agents.filter(a => a.status?.ready).length}
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Pending</CardTitle>
              <Clock className="h-4 w-4 text-yellow-500" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">
                {agents.filter(a => !a.status?.ready && (a.status?.phase === 'Pending' || a.status?.phase === 'Creating')).length}
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Errors</CardTitle>
              <AlertCircle className="h-4 w-4 text-red-500" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">
                {agents.filter(a => !a.status?.ready && a.status?.phase !== 'Pending' && a.status?.phase !== 'Creating').length}
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Agents List */}
        <div className="space-y-4">
          {agents.length === 0 ? (
            <Card>
              <CardContent className="flex items-center justify-center h-64">
                <div className="text-center">
                  <Bot className="h-12 w-12 text-gray-400 mx-auto mb-4" />
                  <h3 className="text-lg font-medium text-gray-900 mb-2">No agents found</h3>
                  <p className="text-gray-600 mb-4">Get started by creating your first language agent.</p>
                  <Button>
                    <Plus className="h-4 w-4 mr-2" />
                    Create Agent
                  </Button>
                </div>
              </CardContent>
            </Card>
          ) : (
            agents.map((agent) => (
              <Card key={agent.metadata.name}>
                <CardHeader>
                  <div className="flex items-start justify-between">
                    <div className="flex items-start space-x-4">
                      <Bot className="h-6 w-6 text-blue-500 mt-1" />
                      <div>
                        <div className="flex items-center space-x-2">
                          <CardTitle className="text-lg">{agent.metadata.name}</CardTitle>
                          <Badge variant="secondary" className="text-xs">
                            {agent.metadata.namespace}
                          </Badge>
                        </div>
                        <CardDescription className="mt-1">
                          {agent.spec.description || 'No description available'}
                        </CardDescription>
                      </div>
                    </div>
                    <div className="flex items-center space-x-2">
                      {getStatusIcon(agent)}
                      <Badge className={getStatusColor(agent)}>
                        {getDisplayStatus(agent)}
                      </Badge>
                    </div>
                  </div>
                </CardHeader>
                <CardContent>
                  <div className="grid gap-4 md:grid-cols-4">
                    <div>
                      <h4 className="text-sm font-medium text-gray-600 mb-1">Model</h4>
                      <p className="text-sm">{agent.spec.model}</p>
                    </div>
                    <div>
                      <h4 className="text-sm font-medium text-gray-600 mb-1">Persona</h4>
                      <p className="text-sm">{agent.spec.persona || 'Not specified'}</p>
                    </div>
                    <div>
                      <h4 className="text-sm font-medium text-gray-600 mb-1">Tools</h4>
                      <div className="flex flex-wrap gap-1">
                        {agent.spec.tools && agent.spec.tools.length > 0 ? (
                          agent.spec.tools.map((tool) => (
                            <Badge key={tool} variant="outline" className="text-xs">
                              {tool}
                            </Badge>
                          ))
                        ) : (
                          <span className="text-xs text-gray-500">No tools configured</span>
                        )}
                      </div>
                    </div>
                    <div>
                      <h4 className="text-sm font-medium text-gray-600 mb-1">Created</h4>
                      <p className="text-sm">{formatTimeAgo(agent.metadata.creationTimestamp)}</p>
                    </div>
                  </div>
                  <div className="flex space-x-2 mt-4">
                    <Button variant="outline" size="sm">
                      Edit
                    </Button>
                    <Button variant="outline" size="sm">
                      View Logs
                    </Button>
                    <Button 
                      variant="destructive" 
                      size="sm"
                      disabled={deleteAgent.isPending}
                      onClick={() => handleDeleteAgent(agent.metadata.name)}
                    >
                      {deleteAgent.isPending ? 'Deleting...' : 'Delete'}
                    </Button>
                  </div>
                </CardContent>
              </Card>
            ))
          )}
        </div>
      </div>
    </AuthenticatedLayout>
  )
}