'use client'

import { useConsole } from '@/contexts/console-context'
import { useState } from 'react'
import { WorkspaceFileTree } from '@/components/workspace/workspace-file-tree'
import { WorkspaceFileViewer } from '@/components/workspace/workspace-file-viewer'
import { WorkspaceToolbar } from '@/components/workspace/workspace-toolbar'
import { useAgent } from '@/hooks/use-agents'
import { FileEntry } from '@/types/workspace'
import { FolderOpen } from 'lucide-react'

export function WorkspacePanel() {
  const { selectedAgent, selectedCluster } = useConsole()
  const [currentPath, setCurrentPath] = useState('/')
  const [selectedFile, setSelectedFile] = useState<FileEntry | null>(null)
  const [refreshTrigger, setRefreshTrigger] = useState(0)

  const { data: agentResponse, isLoading: agentLoading } = useAgent(
    selectedAgent || '',
    selectedCluster || ''
  )

  const handleRefresh = () => {
    setRefreshTrigger((prev) => prev + 1)
  }

  if (!selectedAgent || !selectedCluster) {
    return (
      <div className="flex flex-col items-center justify-center h-full p-6 text-center">
        <FolderOpen className="h-12 w-12 text-stone-400 dark:text-stone-500 mb-3" />
        <p className="text-[11px] font-light text-stone-600 dark:text-stone-400">
          Select an agent to view workspace
        </p>
      </div>
    )
  }

  if (agentLoading || !agentResponse?.data) {
    return (
      <div className="flex flex-col items-center justify-center h-full p-6 text-center">
        <p className="text-[11px] font-light text-stone-600 dark:text-stone-400">
          Loading workspace...
        </p>
      </div>
    )
  }

  return (
    <div className="flex flex-col h-full">
      {/* Header */}
      <div className="border-b border-stone-800/80 dark:border-stone-600/80 py-4 px-4 h-[52px]">
        <h2 className="text-[13px] font-light tracking-widest uppercase text-stone-900 dark:text-stone-300">
          Agent Workspace
        </h2>
      </div>

      {/* Toolbar */}
      <WorkspaceToolbar
        agent={agentResponse.data}
        currentPath={currentPath}
        clusterName={selectedCluster}
        onRefresh={handleRefresh}
        onUploadComplete={handleRefresh}
      />

      {/* File Tree and Viewer */}
      <div className="flex-1 overflow-hidden flex flex-col">
        {!selectedFile ? (
          <WorkspaceFileTree
            agent={agentResponse.data}
            clusterName={selectedCluster}
            currentPath={currentPath}
            onPathChange={setCurrentPath}
            onFileSelect={setSelectedFile}
            refreshTrigger={refreshTrigger}
          />
        ) : (
          <WorkspaceFileViewer
            agent={agentResponse.data}
            clusterName={selectedCluster}
            selectedFile={selectedFile}
            onFileDelete={() => {
              setSelectedFile(null)
              handleRefresh()
            }}
          />
        )}
      </div>
    </div>
  )
}
