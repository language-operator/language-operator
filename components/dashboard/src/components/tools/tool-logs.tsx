'use client'

import { useState, useEffect, useCallback } from 'react'
import { fetchWithOrganization } from '@/lib/api-client'
import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Box } from 'lucide-react'
import { LanguageTool } from '@/types/tool'
import { useLogViewer } from '@/hooks/useLogViewer'
import { PodLogViewer } from '@/components/ui/pod-log-viewer'
import { PodSelector, type PodInfo } from '@/components/ui/pod-selector'

interface ToolLogsProps {
  tool: LanguageTool
  clusterName: string
}

interface PodsResponse {
  data: PodInfo[]
  recommendedPod: string | null
  recommendedContainer: string | null
  deploymentMode: string
  podType: string
}

export function ToolLogs({ tool, clusterName }: ToolLogsProps) {
  const logs = useLogViewer()
  const [pods, setPods] = useState<PodInfo[]>([])
  
  // Placeholder streaming functions for UI consistency
  const startStreaming = () => {
    // TODO: Implement streaming for tool logs
    console.log('Tool streaming not yet implemented')
  }
  
  const stopStreaming = () => {
    // TODO: Implement streaming for tool logs
    console.log('Tool streaming not yet implemented') 
  }
  const [selectedPod, setSelectedPod] = useState<string | null>(null)
  const [deploymentMode, setDeploymentMode] = useState<string>('service')
  const [podType, setPodType] = useState<string>('tool')
  const [podsLoading, setPodsLoading] = useState(true)


  const fetchPods = useCallback(async () => {
    try {
      setPodsLoading(true)
      const response = await fetchWithOrganization(`/api/clusters/${clusterName}/tools/${tool.metadata.name}/pods`)
      if (!response.ok) {
        throw new Error(`Failed to fetch pods: ${response.status} ${response.statusText}`)
      }

      const data: PodsResponse = await response.json()
      setPods(data.data)
      setDeploymentMode(data.deploymentMode)
      setPodType(data.podType)
      
      // Set recommended selections if not already set
      if (!selectedPod && data.recommendedPod) {
        setSelectedPod(data.recommendedPod)
      }
    } catch (err) {
      console.error('Error fetching tool pods:', err)
      logs.setError(err instanceof Error ? err.message : 'Failed to load pods')
    } finally {
      setPodsLoading(false)
    }
  }, [tool.metadata.name, clusterName, selectedPod])

  const fetchLogs = useCallback(async () => {
    if (!selectedPod) return

    try {
      logs.setLoading(true)
      logs.setError(null)

      const url = `/api/clusters/${clusterName}/tools/${tool.metadata.name}/logs?podName=${selectedPod}`

      const response = await fetchWithOrganization(url)
      if (!response.ok) {
        throw new Error(`Failed to fetch logs: ${response.status} ${response.statusText}`)
      }

      const data = await response.json()
      const logLines = data.logs ? data.logs.split('\n').filter((line: string) => line.trim()) : []
      logs.setLogs(logLines)
    } catch (err) {
      console.error('Error fetching tool logs:', err)
      logs.setError(err instanceof Error ? err.message : 'Failed to load logs')
    } finally {
      logs.setLoading(false)
    }
  }, [tool.metadata.name, clusterName, selectedPod])

  useEffect(() => {
    fetchPods()
  }, [fetchPods])

  useEffect(() => {
    if (selectedPod) {
      fetchLogs()
    }
  }, [fetchLogs, selectedPod])

  const getCurrentPod = () => {
    return pods.find(p => p.name === selectedPod)
  }

  const getLogTitle = () => {
    const currentPod = getCurrentPod()
    if (!currentPod) return 'Tool Logs'

    if (deploymentMode === 'sidecar') {
      return `Sidecar Tool Logs (in agent pod)`
    } else {
      return `Tool Service Logs`
    }
  }

  const getLogDescription = () => {
    const currentPod = getCurrentPod()
    if (!currentPod) return `Viewing logs for tool "${tool.metadata.name}"`

    if (deploymentMode === 'sidecar') {
      return `Viewing logs from agent pod "${selectedPod}" using sidecar tool "${tool.metadata.name}"`
    } else {
      return `Viewing logs from ${podType} pod "${selectedPod}" for tool "${tool.metadata.name}"`
    }
  }

  if (podsLoading) {
    return (
      <div className="space-y-6">
        <Card>
          <CardContent className="flex items-center justify-center py-16">
            <div className="text-center">
              <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-gray-900 mx-auto mb-4"></div>
              <p className="text-gray-600">Loading pods...</p>
            </div>
          </CardContent>
        </Card>
      </div>
    )
  }

  if (pods.length === 0) {
    return (
      <div className="space-y-6">
        <Card>
          <CardContent className="flex items-center justify-center py-16">
            <div className="text-center">
              <Box className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
              <h3 className="text-lg font-medium text-foreground mb-2">No Pods Found</h3>
              <p className="text-muted-foreground mb-4">
                {deploymentMode === 'sidecar' 
                  ? `No agent pods found using this sidecar tool.`
                  : `No service pods found for this tool.`
                }
              </p>
              <Button onClick={fetchPods} variant="outline">
                Retry
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {/* Pod Selection */}
      <PodSelector
        pods={pods}
        selectedPod={selectedPod}
        selectedContainer={null}
        deploymentMode={deploymentMode}
        podType={podType}
        onPodChange={setSelectedPod}
        onContainerChange={() => {}} // No container selection needed for tool logs
        onRefresh={fetchPods}
        loading={podsLoading}
        showModeIndicator={true}
        layout="horizontal"
        hideContainerSelection={true}
      />

      {/* Log Viewer */}
      <PodLogViewer
        logs={logs}
        onRefresh={selectedPod ? fetchLogs : undefined}
        refreshDisabled={!selectedPod}
        isStreaming={false}
        onStartStreaming={startStreaming}
        onStopStreaming={stopStreaming}
        streamingDisabled={!selectedPod}
      />
    </div>
  )
}