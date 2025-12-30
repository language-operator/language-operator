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
    <div className="flex items-center justify-between p-4 border-b border-stone-800/80 dark:border-stone-600/80">
      <div className="flex items-center space-x-4">
        <div className="flex items-center space-x-2">
          <FolderOpen className="w-5 h-5 text-muted-foreground" />
          <span className="font-medium">{currentPath}</span>
        </div>
      </div>

      <div className="flex items-center space-x-2">
        {/* Upload Button */}
        <Dialog open={uploadDialogOpen} onOpenChange={setUploadDialogOpen}>
          <DialogTrigger asChild>
            <Button variant="ghost" size="icon" className="h-8 w-8">
              <Upload className="w-4 h-4" />
              <span className="sr-only">Upload files</span>
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
        <Button variant="ghost" size="icon" className="h-8 w-8" onClick={onRefresh}>
          <RefreshCw className="w-4 h-4" />
          <span className="sr-only">Refresh workspace</span>
        </Button>
      </div>
    </div>
  )
}