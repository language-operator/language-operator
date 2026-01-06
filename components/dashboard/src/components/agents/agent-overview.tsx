'use client'

import Link from 'next/link'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { AlertCircle, CheckCircle, Clock } from 'lucide-react'
import { useModels } from '@/hooks/use-models'
import { useTools } from '@/hooks/use-tools'
import { usePersonas } from '@/hooks/use-personas'
import { useOrganization } from '@/components/organization-provider'
import { ResourceEventsActivity } from '@/components/ui/events-activity'
import { LanguageAgent } from '@/types/agent'
import { formatTimeAgo } from './utils'

interface AgentOverviewProps {
  agent: LanguageAgent
  clusterName: string
}

export function AgentOverview({ agent, clusterName }: AgentOverviewProps) {
  const { getOrgUrl } = useOrganization()
  const { data: modelsResponse } = useModels({ clusterName })
  const { data: toolsResponse } = useTools({ clusterName })
  const { data: personasResponse } = usePersonas({ clusterName })

  const allModels = modelsResponse?.data || []
  const allTools = toolsResponse?.data || []
  const allPersonas = personasResponse?.data || []

  // Handle both old format (spec.model) and new format (spec.modelRefs)
  const referencedModel = agent.spec.model?.name
    ? allModels.find((model: any) => model.metadata.name === agent.spec.model?.name)
    : agent.spec.modelRefs?.[0]?.name
    ? allModels.find((model: any) => model.metadata.name === agent.spec.modelRefs?.[0]?.name)
    : null

  // Handle both old format (spec.tools) and new format (spec.toolRefs)
  const referencedTools = agent.spec.tools
    ? agent.spec.tools.map((toolRef) =>
        allTools.find((tool: any) => tool.metadata.name === toolRef.name)
      ).filter(Boolean)
    : agent.spec.toolRefs
    ? agent.spec.toolRefs.map((toolRef) =>
        allTools.find((tool: any) => tool.metadata.name === toolRef.name)
      ).filter(Boolean)
    : []

  // Handle both old format (spec.persona) and new format (spec.personaRefs)
  const referencedPersona = agent.spec.persona?.name
    ? allPersonas.find((persona: any) => persona.metadata.name === agent.spec.persona?.name)
    : agent.spec.personaRefs?.[0]?.name
    ? allPersonas.find((persona: any) => persona.metadata.name === agent.spec.personaRefs?.[0]?.name)
    : null

  return (
    <div className="space-y-6">
      {/* Basic Information - Full Width */}
      <Card>
        <CardHeader>
          <CardTitle>Basic Information</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid gap-4 md:grid-cols-3">
            <div>
              <p className="text-sm font-medium text-stone-600 dark:text-stone-400">Name</p>
              <p className="text-sm">{agent.metadata.name}</p>
            </div>
            <div>
              <p className="text-sm font-medium text-stone-600 dark:text-stone-400">Execution Mode</p>
              <Badge variant="secondary">{agent.spec.executionMode}</Badge>
            </div>
            <div>
              <p className="text-sm font-medium text-stone-600 dark:text-stone-400">Created</p>
              <p className="text-sm">{formatTimeAgo(agent.metadata.creationTimestamp)}</p>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Goal - Prominent Full Width Section */}
      <Card>
        <CardHeader>
          <CardTitle>Goal</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="bg-stone-50 border border-stone-200 p-4 dark:bg-stone-800/50 dark:border-stone-700">
            <p className="text-base leading-relaxed">
              {agent.spec.instructions || 'No goal specified'}
            </p>
          </div>
        </CardContent>
      </Card>

      {/* Models, Tools, and Persona - Third Row */}
      <div className="grid gap-6 md:grid-cols-3">
        <Card>
          <CardHeader>
            <CardTitle>Models</CardTitle>
          </CardHeader>
          <CardContent>
            {agent.spec.model?.name || (agent.spec.modelRefs && agent.spec.modelRefs.length > 0) ? (
              <div className="flex flex-wrap gap-2">
                {/* Handle old format (spec.model) */}
                {agent.spec.model?.name && (
                  <>
                    {referencedModel ? (
                      <Link href={getOrgUrl(`/clusters/${clusterName}/models/${agent.spec.model.name}`)}>
                        <Badge variant="outline" className="hover:bg-primary/10 cursor-pointer">
                          {agent.spec.model.name}
                        </Badge>
                      </Link>
                    ) : (
                      <Badge variant="destructive">
                        {agent.spec.model.name}
                      </Badge>
                    )}
                  </>
                )}
                {/* Handle new format (spec.modelRefs) */}
                {agent.spec.modelRefs?.map((modelRef, index) => {
                  const foundModel = allModels.find((model: any) => model.metadata.name === modelRef.name)
                  return (
                    <div key={index}>
                      {foundModel ? (
                        <Link href={getOrgUrl(`/clusters/${clusterName}/models/${modelRef.name}`)}>
                          <Badge variant="outline" className="hover:bg-primary/10 cursor-pointer">
                            {modelRef.name}
                          </Badge>
                        </Link>
                      ) : (
                        <Badge variant="destructive">
                          {modelRef.name}
                        </Badge>
                      )}
                    </div>
                  )
                })}
              </div>
            ) : (
              <p className="text-sm text-stone-600 dark:text-stone-400">None</p>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Tools</CardTitle>
          </CardHeader>
          <CardContent>
            {(agent.spec.tools && agent.spec.tools.length > 0) || (agent.spec.toolRefs && agent.spec.toolRefs.length > 0) ? (
              <div className="flex flex-wrap gap-2">
                {/* Handle old format (spec.tools) */}
                {agent.spec.tools?.map((toolRef, index) => {
                  const referencedTool = allTools.find((tool: any) => tool.metadata.name === toolRef.name)
                  return (
                    <div key={`old-${index}`}>
                      {referencedTool ? (
                        <Link href={getOrgUrl(`/clusters/${clusterName}/tools/${toolRef.name}`)}>
                          <Badge variant="outline" className="hover:bg-primary/10 cursor-pointer">
                            {toolRef.name}
                          </Badge>
                        </Link>
                      ) : (
                        <Badge variant="destructive">
                          {toolRef.name}
                        </Badge>
                      )}
                    </div>
                  )
                })}
                {/* Handle new format (spec.toolRefs) */}
                {agent.spec.toolRefs?.map((toolRef, index) => {
                  const referencedTool = allTools.find((tool: any) => tool.metadata.name === toolRef.name)
                  return (
                    <div key={`new-${index}`}>
                      {referencedTool ? (
                        <Link href={getOrgUrl(`/clusters/${clusterName}/tools/${toolRef.name}`)}>
                          <Badge variant="outline" className="hover:bg-primary/10 cursor-pointer">
                            {toolRef.name}
                          </Badge>
                        </Link>
                      ) : (
                        <Badge variant="destructive">
                          {toolRef.name}
                        </Badge>
                      )}
                    </div>
                  )
                })}
              </div>
            ) : (
              <p className="text-sm text-stone-600 dark:text-stone-400">None</p>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Persona</CardTitle>
          </CardHeader>
          <CardContent>
            {agent.spec.persona?.name || (agent.spec.personaRefs && agent.spec.personaRefs.length > 0) ? (
              <div className="flex flex-wrap gap-2">
                {/* Handle old format (spec.persona) */}
                {agent.spec.persona?.name && (
                  <>
                    {referencedPersona ? (
                      <Link href={getOrgUrl(`/clusters/${clusterName}/personas/${agent.spec.persona.name}`)}>
                        <Badge variant="outline" className="hover:bg-primary/10 cursor-pointer">
                          {agent.spec.persona.name}
                        </Badge>
                      </Link>
                    ) : (
                      <Badge variant="destructive">
                        {agent.spec.persona.name}
                      </Badge>
                    )}
                  </>
                )}
                {/* Handle new format (spec.personaRefs) */}
                {agent.spec.personaRefs?.map((personaRef, index) => {
                  const foundPersona = allPersonas.find((persona: any) => persona.metadata.name === personaRef.name)
                  return (
                    <div key={index}>
                      {foundPersona ? (
                        <Link href={getOrgUrl(`/clusters/${clusterName}/personas/${personaRef.name}`)}>
                          <Badge variant="outline" className="hover:bg-primary/10 cursor-pointer">
                            {personaRef.name}
                          </Badge>
                        </Link>
                      ) : (
                        <Badge variant="destructive">
                          {personaRef.name}
                        </Badge>
                      )}
                    </div>
                  )
                })}
              </div>
            ) : (
              <p className="text-sm text-stone-600 dark:text-stone-400">None</p>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Status and Conditions */}
      <Card>
        <CardHeader>
          <CardTitle>Status & Conditions</CardTitle>
        </CardHeader>
        <CardContent>
          {agent.status?.conditions && agent.status.conditions.length > 0 ? (
            <div className="space-y-3">
              {agent.status.conditions.map((condition, index) => (
                <div key={index} className="flex items-center justify-between p-3 border border-stone-200 dark:border-stone-700">
                  <div>
                    <p className="text-sm font-medium">{condition.type}</p>
                    {condition.message && (
                      <p className="text-xs text-stone-600 dark:text-stone-400">{condition.message}</p>
                    )}
                  </div>
                  <div className="flex items-center space-x-2">
                    {condition.status === 'True' ? (
                      <CheckCircle className="h-4 w-4 text-green-500" />
                    ) : condition.status === 'False' ? (
                      <AlertCircle className="h-4 w-4 text-red-500" />
                    ) : (
                      <Clock className="h-4 w-4 text-yellow-500" />
                    )}
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <p className="text-sm text-stone-600 dark:text-stone-400">No status conditions available</p>
          )}
        </CardContent>
      </Card>

      {/* Recent Events */}
      <ResourceEventsActivity
        resourceType="agent"
        resourceName={agent.metadata.name || ''}
        namespace={agent.metadata.namespace || ''}
        limit={10}
      />

    </div>
  )
}
