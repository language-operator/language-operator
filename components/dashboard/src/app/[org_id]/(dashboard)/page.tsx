'use client'

import { useState } from 'react'
import Link from 'next/link'
import { ResourceHeader } from '@/components/ui/resource-header'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { ClusterSelectionModal } from '@/components/cluster-selection-modal'
import { EventsActivity } from '@/components/ui/events-activity'
import { Bot, Cpu, Wrench, Users, Boxes, Activity, TrendingUp, Clock, ExternalLink, Settings, Home as HomeIcon } from 'lucide-react'
import { useResourceCounts } from '@/hooks/useResourceCounts'
import { useClusters } from '@/hooks/use-clusters'
import { useAggregatedAgents } from '@/hooks/use-aggregated-agents'
import { ClusterStatusBadge, AgentStatusBadge } from '@/components/ui/resource-status-badge'
import { useRouter } from 'next/navigation'
import { useOrganization } from '@/components/organization-provider'

type QuickActionType = 'agent' | 'model' | 'tool'

export default function OrganizationDashboard() {
  const { getOrgUrl } = useOrganization()
  const { counts, loading, error, refetch } = useResourceCounts()
  const { data: clustersData, isLoading: clustersLoading, error: clustersError } = useClusters({ limit: 5 })
  const { data: agentsData, isLoading: agentsLoading, error: agentsError } = useAggregatedAgents(5)
  const router = useRouter()
  const [modalState, setModalState] = useState<{
    isOpen: boolean
    actionType: QuickActionType | null
  }>({ isOpen: false, actionType: null })

  const hasClusters = !loading && !error && (counts?.clusters || 0) > 0

  const handleQuickAction = (action: string) => {
    switch (action) {
      case 'cluster':
        router.push(getOrgUrl('/clusters/new'))
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
        router.push(getOrgUrl(`/clusters/${clusterName}/agents/new`))
        break
      case 'model':
        router.push(getOrgUrl(`/clusters/${clusterName}/models/new`))
        break
      case 'tool':
        router.push(getOrgUrl(`/clusters/${clusterName}/tools`))
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

  return (
    <div className="space-y-6">
      {/* Standard Resource Header */}
      <ResourceHeader
        icon={HomeIcon}
        title="System Overview"
        subtitle="Monitor and manage your Language Operator resources"
      />

      {/* Stats Grid */}
      <div className="grid gap-4 grid-cols-5">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Clusters</CardTitle>
            <Boxes className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {loading ? '...' : error ? '0' : counts?.clusters || 0}
            </div>
            <p className="text-xs text-muted-foreground">
              {error ? 'Error loading data' : 'LanguageClusters'}
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Agents</CardTitle>
            <Bot className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {loading ? '...' : error ? '0' : counts?.agents || 0}
            </div>
            <p className="text-xs text-muted-foreground">
              {error ? 'Error loading data' : 'LanguageAgents'}
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Tools</CardTitle>
            <Wrench className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {loading ? '...' : error ? '0' : counts?.tools || 0}
            </div>
            <p className="text-xs text-muted-foreground">
              {error ? 'Error loading data' : 'LanguageTools'}
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Models</CardTitle>
            <Cpu className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {loading ? '...' : error ? '0' : counts?.models || 0}
            </div>
            <p className="text-xs text-muted-foreground">
              {error ? 'Error loading data' : 'LanguageModels'}
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Personas</CardTitle>
            <Users className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {loading ? '...' : error ? '0' : counts?.personas || 0}
            </div>
            <p className="text-xs text-muted-foreground">
              {error ? 'Error loading data' : 'LanguagePersonas'}
            </p>
          </CardContent>
        </Card>
      </div>

      {/* Active Resources */}
      <div className="grid gap-6 md:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Boxes className="h-5 w-5" />
              Clusters
            </CardTitle>
            <CardDescription>
              Logical groups of agents, tools, and models
            </CardDescription>
          </CardHeader>
          <CardContent>
            {clustersLoading ? (
              <div className="space-y-3">
                {[1, 2, 3].map((i) => (
                  <div key={i} className="flex items-center justify-between">
                    <div className="flex items-center space-x-3">
                      <div className="h-6 w-6 bg-gray-200 rounded animate-pulse" />
                      <div className="h-4 bg-gray-200 rounded animate-pulse w-24" />
                    </div>
                    <div className="h-3 bg-gray-100 rounded animate-pulse w-16" />
                  </div>
                ))}
              </div>
            ) : clustersError ? (
              <div className="text-center py-4">
                <Boxes className="h-8 w-8 text-gray-400 mx-auto mb-2" />
                <p className="text-sm text-stone-600 dark:text-stone-400">Failed to load clusters</p>
              </div>
            ) : !clustersData?.data || clustersData.data.length === 0 ? (
              <div className="text-center py-4">
                <Boxes className="h-8 w-8 text-gray-400 mx-auto mb-2" />
                <p className="text-sm text-stone-600 dark:text-stone-400">No clusters deployed</p>
                <p className="text-xs text-stone-500 dark:text-stone-500 mt-1">Deploy your first cluster to get started</p>
              </div>
            ) : (
              <div className="space-y-3">
                {clustersData.data.slice(0, 5).map((cluster: any) => (
                  <div key={cluster.metadata.name} className="flex items-center justify-between">
                    <div className="flex items-center space-x-3 min-w-0 flex-1">
                      <div className="min-w-0 flex-1">
                        <Link href={getOrgUrl(`/clusters/${cluster.metadata.name}`)} className="text-sm font-medium truncate hover:text-blue-600 hover:underline">
                          {cluster.metadata.name}
                        </Link>
                        <div className="mt-1">
                          <ClusterStatusBadge cluster={cluster} size="sm" />
                        </div>
                      </div>
                    </div>
                    <div className="flex items-center space-x-1">
                      <Link href={getOrgUrl(`/clusters/${cluster.metadata.name}/agents`)}>
                        <button className="p-1 hover:bg-stone-100 dark:hover:bg-stone-800" title="Agents">
                          <Bot className="h-4 w-4 text-stone-600 dark:text-stone-400" />
                        </button>
                      </Link>
                      <Link href={getOrgUrl(`/clusters/${cluster.metadata.name}/tools`)}>
                        <button className="p-1 hover:bg-stone-100 dark:hover:bg-stone-800" title="Tools">
                          <Wrench className="h-4 w-4 text-stone-600 dark:text-stone-400" />
                        </button>
                      </Link>
                      <Link href={getOrgUrl(`/clusters/${cluster.metadata.name}/models`)}>
                        <button className="p-1 hover:bg-stone-100 dark:hover:bg-stone-800" title="Models">
                          <Cpu className="h-4 w-4 text-stone-600 dark:text-stone-400" />
                        </button>
                      </Link>
                      <Link href={getOrgUrl(`/clusters/${cluster.metadata.name}/personas`)}>
                        <button className="p-1 hover:bg-stone-100 dark:hover:bg-stone-800" title="Personas">
                          <Users className="h-4 w-4 text-stone-600 dark:text-stone-400" />
                        </button>
                      </Link>
                    </div>
                  </div>
                ))}
                {clustersData.data.length > 5 && (
                  <div className="pt-2 border-t">
                    <Link href={getOrgUrl('/clusters')} className="text-xs text-blue-600 hover:text-blue-800">
                      View all {clustersData.data.length} clusters →
                    </Link>
                  </div>
                )}
              </div>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Bot className="h-5 w-5" />
              Agents
            </CardTitle>
            <CardDescription>
              Natural language-based goals and automations
            </CardDescription>
          </CardHeader>
          <CardContent>
            {agentsLoading ? (
              <div className="space-y-3">
                {[1, 2, 3].map((i) => (
                  <div key={i} className="flex items-center justify-between">
                    <div className="flex items-center space-x-3">
                      <div className="h-6 w-6 bg-gray-200 rounded animate-pulse" />
                      <div className="h-4 bg-gray-200 rounded animate-pulse w-24" />
                    </div>
                    <div className="h-3 bg-gray-100 rounded animate-pulse w-16" />
                  </div>
                ))}
              </div>
            ) : agentsError ? (
              <div className="text-center py-4">
                <Bot className="h-8 w-8 text-gray-400 mx-auto mb-2" />
                <p className="text-sm text-stone-600 dark:text-stone-400">Failed to load agents</p>
              </div>
            ) : !agentsData?.data || agentsData.data.length === 0 ? (
              <div className="text-center py-4">
                <Bot className="h-8 w-8 text-gray-400 mx-auto mb-2" />
                <p className="text-sm text-stone-600 dark:text-stone-400">No agents deployed</p>
                <p className="text-xs text-stone-500 dark:text-stone-500 mt-1">Create your first agent to get started</p>
              </div>
            ) : (
              <div className="space-y-3">
                {agentsData.data.slice(0, 5).map((agent: any) => (
                  <div key={agent.metadata.name} className="flex items-center justify-between">
                    <div className="flex items-center space-x-3 min-w-0 flex-1">
                      <div className="min-w-0 flex-1">
                        <Link href={getOrgUrl(`/clusters/${agent.spec?.clusterRef || agent.metadata.namespace}/agents/${agent.metadata.name}`)} className="text-sm font-medium truncate hover:text-blue-600 hover:underline">
                          {agent.metadata.name}
                        </Link>
                        <div className="mt-1">
                          <AgentStatusBadge agent={agent} size="sm" />
                        </div>
                      </div>
                    </div>
                    <div className="flex items-center space-x-1">
                      <Link href={getOrgUrl(`/clusters/${agent.spec?.clusterRef || agent.metadata.namespace}/console?agent=${agent.metadata.name}`)}>
                        <button className="px-3 py-1 text-[10px] tracking-wider uppercase font-light bg-stone-900 text-amber-400 hover:bg-stone-800 dark:bg-amber-400 dark:text-stone-900 dark:hover:bg-amber-500 transition-colors">
                          Connect
                        </button>
                      </Link>
                    </div>
                  </div>
                ))}
                {agentsData.data.length > 5 && (
                  <div className="pt-2 border-t">
                    <Link href={getOrgUrl('/agents')} className="text-xs text-blue-600 hover:text-blue-800">
                      View all {agentsData.data.length} agents →
                    </Link>
                  </div>
                )}
              </div>
            )}
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
              className="flex flex-col items-center p-6 border border-stone-200 hover:bg-stone-50 hover:border-amber-900/40 cursor-pointer transition-all dark:border-stone-600 dark:hover:bg-stone-800 dark:hover:border-amber-600/60"
              onClick={() => handleQuickAction('cluster')}
            >
              <Boxes className="h-8 w-8 text-orange-500 mb-3" />
              <span className="text-sm font-light text-stone-900 dark:text-stone-300">Deploy Cluster</span>
              <span className="text-[11px] font-light text-stone-600 dark:text-stone-400 text-center mt-1">
                Create cluster for agents
              </span>
            </div>
            
            {/* Create Agent - Disabled without clusters */}
            <div 
              className={`flex flex-col items-center p-6 border transition-all ${
                hasClusters 
                  ? 'border-stone-200 hover:bg-stone-50 hover:border-amber-900/40 cursor-pointer dark:border-stone-600 dark:hover:bg-stone-800 dark:hover:border-amber-600/60' 
                  : 'opacity-50 cursor-not-allowed bg-stone-50 border-stone-200 dark:bg-stone-800/50 dark:border-stone-600'
              }`}
              onClick={() => handleQuickAction('agent')}
            >
              <Bot className={`h-8 w-8 mb-3 ${hasClusters ? 'text-blue-500' : 'text-stone-400'}`} />
              <span className={`text-sm font-light ${hasClusters ? 'text-stone-900 dark:text-stone-300' : 'text-stone-400'}`}>
                Create Agent
              </span>
              <span className="text-[11px] font-light text-stone-600 dark:text-stone-400 text-center mt-1">
                {hasClusters ? 'Build a new AI agent' : 'Deploy a cluster first'}
              </span>
            </div>
            
            {/* Add Model - Disabled without clusters */}
            <div 
              className={`flex flex-col items-center p-6 border transition-all ${
                hasClusters 
                  ? 'border-stone-200 hover:bg-stone-50 hover:border-amber-900/40 cursor-pointer dark:border-stone-600 dark:hover:bg-stone-800 dark:hover:border-amber-600/60' 
                  : 'opacity-50 cursor-not-allowed bg-stone-50 border-stone-200 dark:bg-stone-800/50 dark:border-stone-600'
              }`}
              onClick={() => handleQuickAction('model')}
            >
              <Cpu className={`h-8 w-8 mb-3 ${hasClusters ? 'text-green-500' : 'text-stone-400'}`} />
              <span className={`text-sm font-light ${hasClusters ? 'text-stone-900 dark:text-stone-300' : 'text-stone-400'}`}>
                Add Model
              </span>
              <span className="text-[11px] font-light text-stone-600 dark:text-stone-400 text-center mt-1">
                {hasClusters ? 'Connect to LLM provider' : 'Deploy a cluster first'}
              </span>
            </div>
            
            {/* Configure Tool - Disabled without clusters */}
            <div 
              className={`flex flex-col items-center p-6 border transition-all ${
                hasClusters 
                  ? 'border-stone-200 hover:bg-stone-50 hover:border-amber-900/40 cursor-pointer dark:border-stone-600 dark:hover:bg-stone-800 dark:hover:border-amber-600/60' 
                  : 'opacity-50 cursor-not-allowed bg-stone-50 border-stone-200 dark:bg-stone-800/50 dark:border-stone-600'
              }`}
              onClick={() => handleQuickAction('tool')}
            >
              <Wrench className={`h-8 w-8 mb-3 ${hasClusters ? 'text-purple-500' : 'text-stone-400'}`} />
              <span className={`text-sm font-light ${hasClusters ? 'text-stone-900 dark:text-stone-300' : 'text-stone-400'}`}>
                Configure Tool
              </span>
              <span className="text-[11px] font-light text-stone-600 dark:text-stone-400 text-center mt-1">
                {hasClusters ? 'Add capabilities to agents' : 'Deploy a cluster first'}
              </span>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Recent Events */}
      <EventsActivity
        title="Recent Events"
        description="Real-time Kubernetes events across all resources"
        limit={8}
        showNamespace={true}
      />

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
    </div>
  )
}