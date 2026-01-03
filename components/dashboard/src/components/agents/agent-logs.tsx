'use client'

import { useState, useEffect, useRef, useMemo } from 'react'
import { fetchWithOrganization } from '@/lib/api-client'
import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { LanguageAgent } from '@/types/agent'
import { useLogViewer } from '@/hooks/useLogViewer'
import { PodLogViewer } from '@/components/ui/pod-log-viewer'
import { PodSelector, type PodInfo } from '@/components/ui/pod-selector'

interface AgentLogsProps {
  agent: LanguageAgent
  clusterName: string
}

export function AgentLogs({ agent, clusterName }: AgentLogsProps) {
  const logs = useLogViewer()
  const [isStreaming, setIsStreaming] = useState(false)
  const [pods, setPods] = useState<PodInfo[]>([])
  const [selectedPod, setSelectedPod] = useState<string>('')
  const [podsLoading, setPodsLoading] = useState(false)
  const eventSourceRef = useRef<EventSource | null>(null)


  useEffect(() => {
    fetchPods()
    return () => {
      // Cleanup on unmount
      if (eventSourceRef.current) {
        eventSourceRef.current.close()
      }
    }
  }, [agent.metadata.name, clusterName])

  useEffect(() => {
    // Fetch logs when selected pod changes
    if (selectedPod) {
      fetchInitialLogs()
    }
  }, [selectedPod])

  const fetchPods = async () => {
    try {
      setPodsLoading(true)
      logs.setError(null)

      const response = await fetchWithOrganization(`/api/clusters/${clusterName}/agents/${agent.metadata.name}/pods`)
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
  }

  const fetchInitialLogs = async () => {
    try {
      logs.setLoading(true)
      logs.setError(null)

      const url = selectedPod
        ? `/api/clusters/${clusterName}/agents/${agent.metadata.name}/logs?podName=${selectedPod}`
        : `/api/clusters/${clusterName}/agents/${agent.metadata.name}/logs`

      const response = await fetch(url)
      if (!response.ok) {
        throw new Error(`Failed to fetch logs: ${response.status} ${response.statusText}`)
      }

      const data = await response.json()
      const logLines = data.logs ? data.logs.split('\n').filter((line: string) => line.trim()) : []
      logs.setLogs(logLines)
    } catch (err) {
      console.error('Error fetching initial logs:', err)
      logs.setError(err instanceof Error ? err.message : 'Failed to load logs')
    } finally {
      logs.setLoading(false)
    }
  }

  const startStreaming = () => {
    if (eventSourceRef.current) {
      eventSourceRef.current.close()
    }

    setIsStreaming(true)
    logs.setError(null)

    const url = selectedPod
      ? `/api/clusters/${clusterName}/agents/${agent.metadata.name}/logs/stream?podName=${selectedPod}`
      : `/api/clusters/${clusterName}/agents/${agent.metadata.name}/logs/stream`

    const eventSource = new EventSource(url)
    eventSourceRef.current = eventSource

    eventSource.onmessage = (event) => {
      const newLog = event.data
      if (newLog && newLog.trim()) {
        logs.setLogs(prev => [...prev, newLog])
      }
    }

    eventSource.onerror = (error) => {
      console.error('EventSource error:', error)
      logs.setError('Connection lost. Click "Start Streaming" to reconnect.')
      setIsStreaming(false)
      eventSource.close()
    }
  }

  const stopStreaming = () => {
    if (eventSourceRef.current) {
      eventSourceRef.current.close()
      eventSourceRef.current = null
    }
    setIsStreaming(false)
  }




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
        podType="agent"
        onPodChange={(value) => {
          setSelectedPod(value)
          // Stop streaming when switching pods
          if (isStreaming) {
            stopStreaming()
          }
          // Clear existing logs
          logs.clearLogs()
        }}
        onContainerChange={() => {}} // Agent logs don't use container selection
        onRefresh={fetchPods}
        loading={podsLoading}
        layout="horizontal"
      />

      {/* Log Viewer with Streaming Controls */}
      <PodLogViewer
        logs={logs}
        onRefresh={fetchInitialLogs}
        refreshDisabled={!selectedPod}
        isStreaming={isStreaming}
        onStartStreaming={startStreaming}
        onStopStreaming={stopStreaming}
        streamingDisabled={!selectedPod}
      />

      {/* Streaming Status */}
      {isStreaming && (
        <Card>
          <CardContent className="p-4">
            <div className="flex items-center gap-2 text-green-600">
              <div className="animate-pulse w-2 h-2 bg-green-500 rounded-full" />
              <span className="text-sm">Streaming logs in real-time</span>
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  )
}
