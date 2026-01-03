'use client'

import { useState, useEffect, useCallback } from 'react'
import { fetchWithOrganization } from '@/lib/api-client'
import { LanguageModel } from '@/types/model'
import { useLogViewer } from '@/hooks/useLogViewer'
import { PodLogViewer } from '@/components/ui/pod-log-viewer'
import { PodSelector, type PodInfo } from '@/components/ui/pod-selector'

interface ModelLogsProps {
  model: LanguageModel
  clusterName: string
}

export function ModelLogs({ model, clusterName }: ModelLogsProps) {
  const logs = useLogViewer()
  const [pods, setPods] = useState<PodInfo[]>([])
  const [selectedPod, setSelectedPod] = useState<string>('')
  const [podsLoading, setPodsLoading] = useState(false)
  
  // Placeholder streaming functions for UI consistency
  const startStreaming = () => {
    // TODO: Implement streaming for model logs
    console.log('Model streaming not yet implemented')
  }
  
  const stopStreaming = () => {
    // TODO: Implement streaming for model logs  
    console.log('Model streaming not yet implemented')
  }

  const fetchPods = useCallback(async () => {
    try {
      setPodsLoading(true)
      logs.setError(null)

      const response = await fetchWithOrganization(`/api/clusters/${clusterName}/models/${model.metadata.name}/pods`)
      if (!response.ok) {
        throw new Error(`Failed to fetch pods: ${response.status} ${response.statusText}`)
      }

      const data = await response.json()
      setPods(data.data || [])

      // Auto-select the recommended pod
      if (data.recommendedPod && data.data.length > 0) {
        setSelectedPod(data.recommendedPod)
      }
    } catch (err) {
      console.error('Error fetching pods:', err)
      logs.setError(err instanceof Error ? err.message : 'Failed to load pods')
    } finally {
      setPodsLoading(false)
    }
  }, [model.metadata.name, clusterName])

  const fetchLogs = useCallback(async () => {
    if (!selectedPod) return

    try {
      logs.setLoading(true)
      logs.setError(null)

      const url = `/api/clusters/${clusterName}/models/${model.metadata.name}/logs?podName=${selectedPod}`

      const response = await fetchWithOrganization(url)
      if (!response.ok) {
        throw new Error(`Failed to fetch logs: ${response.status} ${response.statusText}`)
      }

      const data = await response.json()
      const logLines = data.logs ? data.logs.split('\n').filter((line: string) => line.trim()) : []
      logs.setLogs(logLines)
    } catch (err) {
      console.error('Error fetching model logs:', err)
      logs.setError(err instanceof Error ? err.message : 'Failed to load logs')
    } finally {
      logs.setLoading(false)
    }
  }, [model.metadata.name, clusterName, selectedPod])

  useEffect(() => {
    fetchPods()
  }, [fetchPods])

  useEffect(() => {
    if (selectedPod) {
      fetchLogs()
    }
  }, [fetchLogs, selectedPod])

  return (
    <div className="space-y-6">
      {/* Pod Selection */}
      <PodSelector
        pods={pods.map(pod => ({
          ...pod,
          isRunning: pod.status === 'Running'
        }))}
        selectedPod={selectedPod}
        selectedContainer={null}
        podType="model"
        onPodChange={(value) => {
          setSelectedPod(value)
          logs.clearLogs()
        }}
        onContainerChange={() => {}} // Model logs don't use container selection
        onRefresh={fetchPods}
        loading={podsLoading}
        layout="horizontal"
      />

      {/* Log Viewer */}
      <PodLogViewer
        logs={logs}
        onRefresh={fetchLogs}
        refreshDisabled={!selectedPod}
        isStreaming={false}
        onStartStreaming={startStreaming}
        onStopStreaming={stopStreaming}
        streamingDisabled={!selectedPod}
      />
    </div>
  )
}