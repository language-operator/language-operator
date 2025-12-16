'use client'

import { useState } from 'react'
import { AuthenticatedLayout } from '@/components/layout/authenticated-layout'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { ClusterSelectionModal } from '@/components/cluster-selection-modal'
import { Bot, Cpu, Wrench, Users, Cloud, Activity, TrendingUp, Clock } from 'lucide-react'
import { useResourceCounts } from '@/hooks/useResourceCounts'
import { useRecentActivity } from '@/hooks/useRecentActivity'
import { useRouter } from 'next/navigation'

type QuickActionType = 'agent' | 'model' | 'tool'

export default function Home() {
  const { counts, loading, error, refetch } = useResourceCounts()
  const { data: recentActivity, isLoading: activityLoading, error: activityError } = useRecentActivity({ limit: 4 })
  const router = useRouter()
  const [modalState, setModalState] = useState<{
    isOpen: boolean
    actionType: QuickActionType | null
  }>({ isOpen: false, actionType: null })

  const hasClusters = !loading && !error && (counts?.clusters || 0) > 0

  const handleQuickAction = (action: string) => {
    switch (action) {
      case 'cluster':
        router.push('/clusters/new')
        break
      case 'agent':
        if (hasClusters) {
          setModalState({ isOpen: true, actionType: 'agent' })
        }
        break
      case 'model':
        if (hasClusters) {
          setModalState({ isOpen: true, actionType: 'model' })
        }
        break
      case 'tool':
        if (hasClusters) {
          setModalState({ isOpen: true, actionType: 'tool' })
        }
        break
    }
  }

  const handleClusterSelect = (clusterName: string) => {
    const { actionType } = modalState
    
    switch (actionType) {
      case 'agent':
        router.push(`/clusters/${clusterName}/agents/new`)
        break
      case 'model':
        router.push(`/clusters/${clusterName}/models/new`)
        break
      case 'tool':
        router.push(`/clusters/${clusterName}/tools/new`)
        break
    }
  }

  const getModalProps = () => {
    switch (modalState.actionType) {
      case 'agent':
        return {
          actionTitle: 'Create Language Agent',
          actionDescription: 'Create a new AI agent to handle tasks and process requests.'
        }
      case 'model':
        return {
          actionTitle: 'Add Language Model',
          actionDescription: 'Connect a new language model provider for your agents to use.'
        }
      case 'tool':
        return {
          actionTitle: 'Configure Tool',
          actionDescription: 'Add new capabilities and tools that agents can use to complete tasks.'
        }
      default:
        return {
          actionTitle: 'Quick Action',
          actionDescription: 'Select a cluster to continue.'
        }
    }
  }

  const getActivityIcon = (type: string, action: string) => {
    const iconClass = "h-5 w-5 mt-0.5"
    
    if (action === 'failed') return <Activity className={`${iconClass} text-red-500`} />
    
    switch (type) {
      case 'agent':
        return <Bot className={`${iconClass} text-blue-500`} />
      case 'model':
        return <Cpu className={`${iconClass} text-green-500`} />
      case 'tool':
        return <Wrench className={`${iconClass} text-purple-500`} />
      case 'persona':
        return <Users className={`${iconClass} text-indigo-500`} />
      case 'cluster':
        return <Cloud className={`${iconClass} text-orange-500`} />
      default:
        return <Activity className={`${iconClass} text-gray-500`} />
    }
  }

  const formatTimestamp = (timestamp: string) => {
    const date = new Date(timestamp)
    const now = new Date()
    const diffInMinutes = Math.floor((now.getTime() - date.getTime()) / (1000 * 60))
    
    if (diffInMinutes < 60) {
      return `${diffInMinutes} minutes ago`
    } else if (diffInMinutes < 1440) { // 24 hours
      const hours = Math.floor(diffInMinutes / 60)
      return `${hours} hour${hours > 1 ? 's' : ''} ago`
    } else {
      const days = Math.floor(diffInMinutes / 1440)
      return `${days} day${days > 1 ? 's' : ''} ago`
    }
  }
  return (
    <AuthenticatedLayout>
      <div className="space-y-6">
        <div>
          <h1 className="text-3xl font-bold font-mono">System Overview</h1>
          <p className="text-gray-600 mt-2">
            Monitor and manage your Language Operator resources
          </p>
        </div>

        {/* Stats Grid */}
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Language Agents</CardTitle>
              <Bot className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">
                {loading ? '...' : error ? '0' : counts?.agents || 0}
              </div>
              <p className="text-xs text-muted-foreground">
                {error ? 'Error loading data' : 'Active in your namespace'}
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Language Models</CardTitle>
              <Cpu className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">
                {loading ? '...' : error ? '0' : counts?.models || 0}
              </div>
              <p className="text-xs text-muted-foreground">
                {error ? 'Error loading data' : 'Available for agents'}
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Language Tools</CardTitle>
              <Wrench className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">
                {loading ? '...' : error ? '0' : counts?.tools || 0}
              </div>
              <p className="text-xs text-muted-foreground">
                {error ? 'Error loading data' : 'Ready for agents'}
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Language Personas</CardTitle>
              <Users className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">
                {loading ? '...' : error ? '0' : counts?.personas || 0}
              </div>
              <p className="text-xs text-muted-foreground">
                {error ? 'Error loading data' : 'Personality templates'}
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Language Clusters</CardTitle>
              <Cloud className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">
                {loading ? '...' : error ? '0' : counts?.clusters || 0}
              </div>
              <p className="text-xs text-muted-foreground">
                {error ? 'Error loading data' : 'Deployed configurations'}
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">System Health</CardTitle>
              <Activity className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">99.2%</div>
              <p className="text-xs text-muted-foreground">
                30-day uptime
              </p>
            </CardContent>
          </Card>
        </div>

        {/* Activity and Quick Actions */}
        <div className="grid gap-6 md:grid-cols-2">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Clock className="h-5 w-5" />
                Recent Activity
              </CardTitle>
              <CardDescription>
                Latest changes to your resources
              </CardDescription>
            </CardHeader>
            <CardContent>
              {activityLoading ? (
                <div className="space-y-4">
                  {[1, 2, 3, 4].map((i) => (
                    <div key={i} className="flex items-start space-x-3">
                      <div className="h-5 w-5 bg-gray-200 rounded mt-0.5 animate-pulse" />
                      <div className="flex-1 min-w-0">
                        <div className="h-4 bg-gray-200 rounded animate-pulse mb-1" />
                        <div className="h-3 bg-gray-100 rounded animate-pulse w-3/4" />
                      </div>
                    </div>
                  ))}
                </div>
              ) : activityError ? (
                <div className="text-center py-8">
                  <Activity className="h-8 w-8 text-gray-400 mx-auto mb-2" />
                  <p className="text-sm text-gray-500">Failed to load recent activity</p>
                </div>
              ) : !recentActivity?.data || recentActivity.data.length === 0 ? (
                <div className="text-center py-8">
                  <Clock className="h-8 w-8 text-gray-400 mx-auto mb-2" />
                  <p className="text-sm text-gray-500">No recent activity</p>
                  <p className="text-xs text-gray-400 mt-1">Activity will appear here when you start using your resources</p>
                </div>
              ) : (
                <div className="space-y-4">
                  {recentActivity.data.map((activity) => (
                    <div key={activity.id} className="flex items-start space-x-3">
                      {getActivityIcon(activity.type, activity.action)}
                      <div className="flex-1 min-w-0">
                        <p className="text-sm font-medium text-gray-900">
                          {activity.message}
                        </p>
                        <p className="text-sm text-gray-500">
                          {formatTimestamp(activity.timestamp)}
                          {activity.namespace && activity.namespace !== 'default' && (
                            <span className="text-gray-400"> in {activity.namespace} namespace</span>
                          )}
                        </p>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <TrendingUp className="h-5 w-5" />
                System Status
              </CardTitle>
              <CardDescription>
                Current health of your infrastructure
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="space-y-3">
                <div className="flex items-center justify-between">
                  <span className="text-sm font-medium">API Gateway</span>
                  <div className="flex items-center space-x-2">
                    <div className="h-2 w-2 rounded-full bg-green-500"></div>
                    <span className="text-sm text-green-600">Healthy</span>
                  </div>
                </div>

                <div className="flex items-center justify-between">
                  <span className="text-sm font-medium">PostgreSQL Database</span>
                  <div className="flex items-center space-x-2">
                    <div className="h-2 w-2 rounded-full bg-green-500"></div>
                    <span className="text-sm text-green-600">Connected</span>
                  </div>
                </div>

                <div className="flex items-center justify-between">
                  <span className="text-sm font-medium">Language Clusters</span>
                  <div className="flex items-center space-x-2">
                    <div className="h-2 w-2 rounded-full bg-yellow-500"></div>
                    <span className="text-sm text-yellow-600">Scaling</span>
                  </div>
                </div>

                <div className="flex items-center justify-between">
                  <span className="text-sm font-medium">Kubernetes API</span>
                  <div className="flex items-center space-x-2">
                    <div className="h-2 w-2 rounded-full bg-green-500"></div>
                    <span className="text-sm text-green-600">Available</span>
                  </div>
                </div>

                <div className="flex items-center justify-between">
                  <span className="text-sm font-medium">Monitoring</span>
                  <div className="flex items-center space-x-2">
                    <div className="h-2 w-2 rounded-full bg-green-500"></div>
                    <span className="text-sm text-green-600">Active</span>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Quick Actions */}
        <Card>
          <CardHeader>
            <CardTitle>Quick Actions</CardTitle>
            <CardDescription>
              Get started with common tasks
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
              {/* Deploy Cluster - Always enabled, shown first */}
              <div 
                className="flex flex-col items-center p-4 border rounded-lg hover:bg-gray-50 cursor-pointer transition-colors"
                onClick={() => handleQuickAction('cluster')}
              >
                <Cloud className="h-8 w-8 text-orange-500 mb-2" />
                <span className="text-sm font-medium">Deploy Cluster</span>
                <span className="text-xs text-gray-500 text-center mt-1">
                  Create cluster for agents
                </span>
              </div>
              
              {/* Create Agent - Disabled without clusters */}
              <div 
                className={`flex flex-col items-center p-4 border rounded-lg transition-colors ${
                  hasClusters 
                    ? 'hover:bg-gray-50 cursor-pointer' 
                    : 'opacity-50 cursor-not-allowed bg-gray-50'
                }`}
                onClick={() => handleQuickAction('agent')}
              >
                <Bot className={`h-8 w-8 mb-2 ${hasClusters ? 'text-blue-500' : 'text-gray-400'}`} />
                <span className={`text-sm font-medium ${hasClusters ? '' : 'text-gray-400'}`}>
                  Create Agent
                </span>
                <span className="text-xs text-gray-500 text-center mt-1">
                  {hasClusters ? 'Build a new AI agent' : 'Deploy a cluster first'}
                </span>
              </div>
              
              {/* Add Model - Disabled without clusters */}
              <div 
                className={`flex flex-col items-center p-4 border rounded-lg transition-colors ${
                  hasClusters 
                    ? 'hover:bg-gray-50 cursor-pointer' 
                    : 'opacity-50 cursor-not-allowed bg-gray-50'
                }`}
                onClick={() => handleQuickAction('model')}
              >
                <Cpu className={`h-8 w-8 mb-2 ${hasClusters ? 'text-green-500' : 'text-gray-400'}`} />
                <span className={`text-sm font-medium ${hasClusters ? '' : 'text-gray-400'}`}>
                  Add Model
                </span>
                <span className="text-xs text-gray-500 text-center mt-1">
                  {hasClusters ? 'Connect to LLM provider' : 'Deploy a cluster first'}
                </span>
              </div>
              
              {/* Configure Tool - Disabled without clusters */}
              <div 
                className={`flex flex-col items-center p-4 border rounded-lg transition-colors ${
                  hasClusters 
                    ? 'hover:bg-gray-50 cursor-pointer' 
                    : 'opacity-50 cursor-not-allowed bg-gray-50'
                }`}
                onClick={() => handleQuickAction('tool')}
              >
                <Wrench className={`h-8 w-8 mb-2 ${hasClusters ? 'text-purple-500' : 'text-gray-400'}`} />
                <span className={`text-sm font-medium ${hasClusters ? '' : 'text-gray-400'}`}>
                  Configure Tool
                </span>
                <span className="text-xs text-gray-500 text-center mt-1">
                  {hasClusters ? 'Add capabilities to agents' : 'Deploy a cluster first'}
                </span>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Cluster Selection Modal */}
      {modalState.actionType && (
        <ClusterSelectionModal
          isOpen={modalState.isOpen}
          onClose={() => setModalState({ isOpen: false, actionType: null })}
          onClusterSelect={handleClusterSelect}
          actionType={modalState.actionType}
          {...getModalProps()}
        />
      )}
    </AuthenticatedLayout>
  )
}
