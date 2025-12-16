'use client'

import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger, DialogFooter } from '@/components/ui/dialog'
import { Badge } from '@/components/ui/badge'
import { LanguageAgent } from '@/types/agent'
import { Upload, RefreshCw, FolderOpen, Info } from 'lucide-react'
import { useToast } from '@/hooks/use-toast'

interface WorkspaceToolbarProps {
  agent: LanguageAgent
  clusterName: string
  currentPath: string
  onRefresh: () => void
  onUploadComplete: () => void
}

export function WorkspaceToolbar({ 
  agent, 
  clusterName, 
  currentPath, 
  onRefresh, 
  onUploadComplete 
}: WorkspaceToolbarProps) {
  const [isUploading, setIsUploading] = useState(false)
  const [uploadDialogOpen, setUploadDialogOpen] = useState(false)
  const { toast } = useToast()

  const handleFileUpload = async (event: React.ChangeEvent<HTMLInputElement>) => {
    const files = event.target.files
    if (!files || files.length === 0) return

    setIsUploading(true)
    
    try {
      for (const file of files) {
        const formData = new FormData()
        formData.append('file', file)
        formData.append('path', currentPath)

        const response = await fetch(
          `/api/clusters/${clusterName}/agents/${agent.metadata?.name}/workspace/files`,
          {
            method: 'POST',
            body: formData,
          }
        )

        if (!response.ok) {
          const error = await response.json()
          throw new Error(error.error || 'Upload failed')
        }
      }

      toast({
        title: 'Upload successful',
        description: `Uploaded ${files.length} file(s) to ${currentPath}`,
      })

      onUploadComplete()
      setUploadDialogOpen(false)
    } catch (error: any) {
      toast({
        title: 'Upload failed',
        description: error.message,
        variant: 'destructive',
      })
    } finally {
      setIsUploading(false)
    }
  }

  const workspaceInfo = agent.spec?.workspace
  const mountPath = workspaceInfo?.mountPath || '/workspace'
  const size = workspaceInfo?.size || '10Gi'
  const accessMode = workspaceInfo?.accessMode || 'ReadWriteOnce'

  return (
    <div className="flex items-center justify-between">
      <div className="flex items-center space-x-4">
        <div className="flex items-center space-x-2">
          <FolderOpen className="w-5 h-5 text-muted-foreground" />
          <span className="font-medium">{currentPath}</span>
        </div>
      </div>

      <div className="flex items-center space-x-2">
        {/* Workspace Info */}
        <Dialog>
          <DialogTrigger asChild>
            <Button variant="ghost" size="sm">
              <Info className="w-4 h-4 mr-2" />
              Info
            </Button>
          </DialogTrigger>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>Workspace Information</DialogTitle>
            </DialogHeader>
            <div className="space-y-4">
              <div>
                <label className="text-sm font-medium">Mount Path</label>
                <p className="text-sm text-muted-foreground">{mountPath}</p>
              </div>
              <div>
                <label className="text-sm font-medium">Storage Size</label>
                <p className="text-sm text-muted-foreground">{size}</p>
              </div>
              <div>
                <label className="text-sm font-medium">Access Mode</label>
                <p className="text-sm text-muted-foreground">{accessMode}</p>
              </div>
              <div>
                <label className="text-sm font-medium">PVC Name</label>
                <p className="text-sm text-muted-foreground">{agent.metadata?.name}-workspace</p>
              </div>
              {workspaceInfo?.storageClassName && (
                <div>
                  <label className="text-sm font-medium">Storage Class</label>
                  <p className="text-sm text-muted-foreground">{workspaceInfo.storageClassName}</p>
                </div>
              )}
            </div>
          </DialogContent>
        </Dialog>

        {/* Upload Button */}
        <Dialog open={uploadDialogOpen} onOpenChange={setUploadDialogOpen}>
          <DialogTrigger asChild>
            <Button variant="outline" size="sm">
              <Upload className="w-4 h-4 mr-2" />
              Upload
            </Button>
          </DialogTrigger>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>Upload Files</DialogTitle>
            </DialogHeader>
            <div className="space-y-4">
              <div>
                <label className="text-sm font-medium">Upload to: {currentPath}</label>
              </div>
              <Input
                type="file"
                multiple
                onChange={handleFileUpload}
                disabled={isUploading}
                className="cursor-pointer"
              />
              <p className="text-xs text-muted-foreground">
                Maximum file size: 100MB per file. Supported formats: text files, images, documents.
              </p>
            </div>
            <DialogFooter>
              <Button
                variant="outline"
                onClick={() => setUploadDialogOpen(false)}
                disabled={isUploading}
              >
                Cancel
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>

        {/* Refresh Button */}
        <Button variant="outline" size="sm" onClick={onRefresh}>
          <RefreshCw className="w-4 h-4 mr-2" />
          Refresh
        </Button>
      </div>
    </div>
  )
}