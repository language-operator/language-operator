'use client'

import { useState } from 'react'
import { LanguageAgent } from '@/types/agent'
import { Card, CardContent } from '@/components/ui/card'
import { WorkspaceFileTree } from './workspace-file-tree'
import { WorkspaceFileViewer } from './workspace-file-viewer'
import { WorkspaceToolbar } from './workspace-toolbar'
import { FileEntry } from '@/types/workspace'

interface AgentWorkspaceProps {
  agent: LanguageAgent
  clusterName: string
}

export function AgentWorkspace({ agent, clusterName }: AgentWorkspaceProps) {
  const [currentPath, setCurrentPath] = useState('/')
  const [selectedFile, setSelectedFile] = useState<FileEntry | null>(null)
  const [refreshTrigger, setRefreshTrigger] = useState(0)

  const handleFileSelect = (file: FileEntry) => {
    if (file.type === 'file') {
      setSelectedFile(file)
    } else {
      // Navigate to directory
      setCurrentPath(file.path)
      setSelectedFile(null)
    }
  }

  const handleRefresh = () => {
    setRefreshTrigger(prev => prev + 1)
  }

  const handleUploadComplete = () => {
    handleRefresh()
  }

  return (
    <div className="flex-1 flex flex-col space-y-4">
      {/* Toolbar */}
      <WorkspaceToolbar
        agent={agent}
        clusterName={clusterName}
        currentPath={currentPath}
        onRefresh={handleRefresh}
        onUploadComplete={handleUploadComplete}
      />

      {/* Main workspace area */}
      <div className="flex-1 flex gap-4 min-h-0">
        {/* File tree sidebar */}
        <Card className="w-80 flex flex-col pt-1">
          <CardContent className="p-4 flex-1 overflow-hidden">
            <WorkspaceFileTree
              agent={agent}
              clusterName={clusterName}
              currentPath={currentPath}
              onFileSelect={handleFileSelect}
              onPathChange={setCurrentPath}
              refreshTrigger={refreshTrigger}
            />
          </CardContent>
        </Card>

        {/* File viewer */}
        <Card className="flex-1 flex flex-col min-w-0 pt-1">
          <CardContent className="p-0 flex-1 overflow-hidden">
            <WorkspaceFileViewer
              agent={agent}
              clusterName={clusterName}
              selectedFile={selectedFile}
              onFileDelete={handleRefresh}
              onBack={() => setSelectedFile(null)}
            />
          </CardContent>
        </Card>
      </div>
    </div>
  )
}